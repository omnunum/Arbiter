package main

import (
	"fmt"
	"github.com/go-redis/redis"
	"github.com/pkg/errors"
	tb "gopkg.in/tucnak/telebot.v2"
)

func userHasAdminManagementAccess(userID int, chatID int) (bool, error) {
	owner, err := R.Get(fmt.Sprintf("chat:%d:owner", chatID)).Int64()
	if err != redis.Nil {
		return int(owner) == userID, nil
	}
	return false, err
}

func getUsersActiveChat(userID int) (int, string, error) {
	key := fmt.Sprintf("user:%d:activeChat", userID)
	activeChatID, err := R.Get(key).Int64()
	if err != nil {
		errors.Wrapf(err, "userID %d doesn't have an active chat but tried to access it", userID)
		return 0, "", err
	}
	chatName, err := getChatTitle(int(activeChatID))
	return int(activeChatID), chatName, nil
}

func getChatTitle(chatID int) (string, error) {
	key := fmt.Sprintf("chat:%d:title", chatID)
	if title, err := R.Get(key).Result(); err == redis.Nil {
		errors.Wrapf(err, "could not access title for chat %d", chatID)
		return "", err
	} else {
		return title, nil
	}
}

func getUserName(userID int) (string, error) {
	key := fmt.Sprintf("user:%d:info", userID)
	if data, err := R.Get(key).Bytes(); err == redis.Nil {
		errors.Wrapf(err, "could not access title for chat %d", userID)
		return "", err
	} else {
		u := DecodeUser(data)
		if u.Username != "" {
			return u.Username, nil
		} else {
			name := fmt.Sprintf("%s %s", u.FirstName, u.LastName)
			return name, nil
		}
	}
}

func updateChatAdmins(chatID int) error {
	if members, err := B.AdminsOf(&tb.Chat{ID: int64(chatID)}); err != nil {
		LogE.Printf("error fetching admins for chat %s", chatID)
		return err
	} else {
		for _, u := range members {
			// update all info for chat admins in case any have changed
			R.Set(fmt.Sprintf("user:%d:info", u.User.ID),
				EncodeUser(u.User), 0)
			R.SAdd(fmt.Sprintf("chat:%d:admins", chatID), u.User.ID)
		}
	}
	return nil
}
