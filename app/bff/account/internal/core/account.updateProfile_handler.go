// Copyright 2022 Teamgram Authors
//  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Author: teamgramio (teamgram.io@gmail.com)
//

package core

import (
	"github.com/teamgram/proto/mtproto"
	"github.com/teamgram/teamgram-server/app/messenger/sync/sync"
	userpb "github.com/teamgram/teamgram-server/app/service/biz/user/user"
)

// AccountUpdateProfile
// account.updateProfile#78515775 flags:# first_name:flags.0?string last_name:flags.1?string about:flags.2?string = User;
func (c *AccountCore) AccountUpdateProfile(in *mtproto.TLAccountUpdateProfile) (*mtproto.User, error) {
	me, err := c.svcCtx.Dao.UserClient.UserGetImmutableUser(c.ctx, &userpb.TLUserGetImmutableUser{
		Id: c.MD.UserId,
	})
	if err != nil {
		c.Logger.Errorf("account.updateProfile - error getting user: %v", err)
		return nil, err
	}

	firstName := in.GetFirstName()
	lastName := in.GetLastName()
	about := in.GetAbout()

	c.Logger.Debugf("account.updateProfile - first name: %s, last name: %s, about: %s", firstName, lastName, about)

	if firstName != nil || lastName != nil {
		// Both first name and last name must be provided
		if firstName == nil || lastName == nil {
			err = mtproto.ErrFirstnameInvalid
			c.Logger.Errorf("account.updateProfile - error: bad request (%v)", err)
			return nil, err
		}

		if err = updateFirstNameAndLastName(firstName.GetValue(), lastName.GetValue(), c, me); err != nil {
			return nil, err
		}
	}

	if about != nil {
		if err = updateAbout(about.GetValue(), c, me); err != nil {
			return nil, err
		}
	}

	c.Logger.Debugf("account.updateProfile - success, first name: %s, last name: %s, about: %s", me.FirstName(), me.LastName(), me.About())
	return me.ToSelfUser(), nil
}

func updateAbout(aboutValue string, c *AccountCore, me *mtproto.ImmutableUser) error {
	if len(aboutValue) > 70 {
		err := mtproto.ErrAboutTooLong
		c.Logger.Errorf("account.updateProfile - error: %v", err)
		return err
	}

	if aboutValue != me.About() {
		c.Logger.Debugf("account.updateProfile - updating about to %s", aboutValue)
		if _, err := c.svcCtx.Dao.UserClient.UserUpdateAbout(c.ctx, &userpb.TLUserUpdateAbout{
			UserId: c.MD.UserId,
			About:  aboutValue,
		}); err != nil {
			c.Logger.Errorf("account.updateProfile - error updating about: %v", err)
			return err
		}
		me.SetAbout(aboutValue)
	} else {
		c.Logger.Debugf("account.updateProfile - about is the same, not updating")
	}

	return nil
}

func updateFirstNameAndLastName(firstName string, lastName string, c *AccountCore, me *mtproto.ImmutableUser) error {
	if firstName == "" {
		err := mtproto.ErrFirstnameInvalid
		c.Logger.Errorf("account.updateProfile - error: bad request (%v)", err)
		return err
	}

	if firstName != me.FirstName() || lastName != me.LastName() {
		c.Logger.Debugf("account.updateProfile - updating first name to %s and last name to %s", firstName, lastName)
		if _, err := c.svcCtx.Dao.UserClient.UserUpdateFirstAndLastName(c.ctx, &userpb.TLUserUpdateFirstAndLastName{
			UserId:    c.MD.UserId,
			FirstName: firstName,
			LastName:  lastName,
		}); err != nil {
			c.Logger.Errorf("account.updateProfile - error updating names: %v", err)
			return err
		}

		me.SetFirstName(firstName)
		me.SetLastName(lastName)

		if _, err := c.svcCtx.Dao.SyncClient.SyncUpdatesNotMe(c.ctx, &sync.TLSyncUpdatesNotMe{
			UserId:        c.MD.UserId,
			PermAuthKeyId: c.MD.PermAuthKeyId,
			Updates: mtproto.MakeUpdatesByUpdates(mtproto.MakeTLUpdateUserName(&mtproto.Update{
				UserId:    c.MD.UserId,
				FirstName: firstName,
				LastName:  lastName,
				Username:  me.Username(),
			}).To_Update()),
		}); err != nil {
			c.Logger.Errorf("account.updateProfile - error syncing updates: %v", err)
			return err
		}
	} else {
		c.Logger.Debugf("account.updateProfile - names are the same, not updating")
	}

	return nil
}
