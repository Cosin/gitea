// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"crypto/md5"
	"fmt"
	"image/png"
	"io"
	"strings"

	"code.gitea.io/gitea/models/avatars"
	"code.gitea.io/gitea/models/db"
	system_model "code.gitea.io/gitea/models/system"
	"code.gitea.io/gitea/modules/avatar"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/storage"
)

// CustomAvatarRelativePath returns user custom avatar relative path.
func (u *User) CustomAvatarRelativePath() string {
	return u.Avatar
}

// GenerateRandomAvatar generates a random avatar for user.
func GenerateRandomAvatar(ctx context.Context, u *User) error {
	seed := u.Email
	if len(seed) == 0 {
		seed = u.Name
	}

	img, err := avatar.RandomImage([]byte(seed))
	if err != nil {
		return fmt.Errorf("RandomImage: %w", err)
	}

	u.Avatar = avatars.HashEmail(seed)

	// Don't share the images so that we can delete them easily
	if err := storage.SaveFrom(storage.Avatars, u.CustomAvatarRelativePath(), func(w io.Writer) error {
		if err := png.Encode(w, img); err != nil {
			log.Error("Encode: %v", err)
		}
		return err
	}); err != nil {
		return fmt.Errorf("Failed to create dir %s: %w", u.CustomAvatarRelativePath(), err)
	}

	if _, err := db.GetEngine(ctx).ID(u.ID).Cols("avatar").Update(u); err != nil {
		return err
	}

	log.Info("New random avatar created: %d", u.ID)
	return nil
}

// AvatarLinkWithSize returns a link to the user's avatar with size. size <= 0 means default size
func (u *User) AvatarLinkWithSize(size int) string {
	if u.ID == -1 {
		// ghost user
		return avatars.DefaultAvatarLink()
	}

	useLocalAvatar := false
	autoGenerateAvatar := false

	disableGravatarSetting, _ := system_model.GetSetting(system_model.KeyPictureDisableGravatar)

	disableGravatar := disableGravatarSetting.GetValueBool()

	switch {
	case u.UseCustomAvatar:
		useLocalAvatar = true
	case disableGravatar, setting.OfflineMode:
		useLocalAvatar = true
		autoGenerateAvatar = true
	}

	if useLocalAvatar {
		if u.Avatar == "" && autoGenerateAvatar {
			if err := GenerateRandomAvatar(db.DefaultContext, u); err != nil {
				log.Error("GenerateRandomAvatar: %v", err)
			}
		}
		if u.Avatar == "" {
			return avatars.DefaultAvatarLink()
		}
		return avatars.GenerateUserAvatarImageLink(u.Avatar, size)
	}
	return avatars.GenerateEmailAvatarFastLink(u.AvatarEmail, size)
}

// AvatarLink returns the full avatar link with http host
func (u *User) AvatarLink() string {
	link := u.AvatarLinkWithSize(0)
	if !strings.HasPrefix(link, "//") && !strings.Contains(link, "://") {
		return setting.AppURL + strings.TrimPrefix(link, setting.AppSubURL+"/")
	}
	return link
}

// IsUploadAvatarChanged returns true if the current user's avatar would be changed with the provided data
func (u *User) IsUploadAvatarChanged(data []byte) bool {
	if !u.UseCustomAvatar || len(u.Avatar) == 0 {
		return true
	}
	avatarID := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%d-%x", u.ID, md5.Sum(data)))))
	return u.Avatar != avatarID
}

// ExistsWithAvatarAtStoragePath returns true if there is a user with this Avatar
func ExistsWithAvatarAtStoragePath(ctx context.Context, storagePath string) (bool, error) {
	// See func (u *User) CustomAvatarRelativePath()
	// u.Avatar is used directly as the storage path - therefore we can check for existence directly using the path
	return db.GetEngine(ctx).Where("`avatar`=?", storagePath).Exist(new(User))
}
