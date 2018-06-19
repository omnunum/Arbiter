package main

import (
		"strings"
	"fmt"
	tb "gopkg.in/tucnak/telebot.v2"
	"github.com/go-redis/redis"
	)

func userHasAdminManagementAccess(userID int, chatID int) (bool, error) {
	owner, err := R.Get(fmt.Sprintf("chat:%d:owner", chatID)).Int64()
	if err != redis.Nil{
		return int(owner) == userID, nil
	}
	return false, err
}

func getUsersActiveChat(userID int) (int, string, error) {
	key := fmt.Sprintf("user:%d:activeChat", userID)
	activeChatID, err := R.Get(key).Int64()
	if err != nil {
		LogE.Printf("userID %d doesn't have an active chat but tried to access it", userID)
		return 0, "", err
	}
	chatName, err := getChatTitle(int(activeChatID))
	return int(activeChatID), chatName, nil
}

func getChatTitle(chatID int) (string, error) {
	key := fmt.Sprintf("chat:%d:title", chatID)
	if title, err := R.Get(key).Result(); err == redis.Nil {
		LogE.Printf("could not access title for chat %d", chatID)
		return "", err
	} else {
		return title, nil
	}
}

func getReplyKeyboardForCommands(commands []FunctionButton) ([][]tb.ReplyButton) {
	keys := [][]tb.ReplyButton{}
	row := []tb.ReplyButton{}
	for _, v := range commands {
		button := tb.ReplyButton{Text: v.Label}
		B.Handle(&button, v.Function)
		row = append(row, button)
	}
	keys = append(keys, row)
	return keys
}

func getInlineKeyboardForCommands(commands []FunctionButton) ([][]tb.InlineButton) {
	keys := [][]tb.InlineButton{}
	row := []tb.InlineButton{}
	for _, v := range commands {
		button := tb.InlineButton{Text: v.Label}
		B.Handle(&button, v)
		row = append(row, button)
	}
	keys = append(keys, row)
	return keys
}

func getInlineButtonForMessages(buttonTexts []string) ([][]tb.InlineButton) {
	keys := [][]tb.InlineButton{}
	row := []tb.InlineButton{}
	for _, t := range buttonTexts {
		button := tb.InlineButton{
			Unique: "",
			Text:   t}
		B.Handle(&button, t)
		row = append(row, button)
	}
	keys = append(keys, row)
	return keys
}

func addAdmin(ms []*tb.Message) {
	m := ms[0]
	split := strings.Split(strings.Replace(m.Text, "/addAdmin ", "", 1), " ")
	adminName := split[0]
	chatID, _, _ := getUsersActiveChat(m.Sender.ID)
	setKey := fmt.Sprintf("chat:%d:admins", chatID)
	R.SAdd(setKey, adminName)
	val, _ := R.SMembers(setKey).Result()
	B.Send(m.Sender, fmt.Sprintf("admins now %s", val))
}

func removeAdmin(ms []*tb.Message) {
	m := ms[0]
	split := strings.Split(strings.Replace(m.Text, "/removeAdmin ", "", 1), " ")
	adminName := split[0]
	chatID, chanTitle, _ := getUsersActiveChat(m.Sender.ID)
	setKey := fmt.Sprintf("chat:%d:admins", chatID)
	R.SRem(setKey, adminName)
	val, _ := R.SMembers(setKey).Result()
	B.Send(m.Sender, fmt.Sprintf("admins for chat %s now %s", chanTitle, val))
}

func viewAdmins(ms []*tb.Message) {
	m := ms[0]
	chatID, chanTitle, _ := getUsersActiveChat(m.Sender.ID)
	setKey := fmt.Sprintf("chat:%d:admins", chatID)
	val, _ := R.SMembers(setKey).Result()
	B.Send(m.Sender, fmt.Sprintf("admins for chat %s now %s", chanTitle, val))
}

func addCommand(ms []*tb.Message) {
	sender := ms[0].Sender
	commandName := ms[0].Text
	if ! strings.HasPrefix(commandName, "/") {
		commandName = "/" + commandName
	}
	commandText := ms[1].Text
	LogI.Printf("entered addCommand with msgs [%s %s]", commandName, commandText)
	if len(ms) == 1 {
		B.Send(sender, fmt.Sprint(
			"you need to specify a command and response to add, such as /addCommand commandName;response text"))
	}
	if err := registerStaticCommand(sender.ID, commandName, commandText); err != nil {
		msg := fmt.Sprintf("error while trying to add command %s", commandName)
		B.Send(sender, msg)
		LogE.Printf(fmt.Sprintf("%s: %s"), msg, err)
	}
	B.Send(sender, fmt.Sprintf("added/updated command %s", commandName))
}

func removeCommand(ms []*tb.Message) {
	m := ms[0]
	LogI.Printf("entered removeCommand with msg %s", m.Text)
	commandName := strings.Replace(m.Text, "/removeCommand ", "", 1)
	if ! strings.HasPrefix(commandName, "/") {
		commandName = "/" + commandName
	}
	if err := unregisterStaticCommand(m.Sender.ID, commandName); err != nil {
		msg := fmt.Sprintf("error while trying to remove command %s", commandName)
		B.Send(m.Sender, msg)
		LogE.Printf(fmt.Sprintf("%s: %s"), msg, err)
	}
	B.Send(m.Sender, fmt.Sprintf("removed command %s", commandName))
}

func viewCommands(ms []*tb.Message) {
	m := ms[0]
	chanID, chanTitle, _ := getUsersActiveChat(m.Sender.ID)
	key := fmt.Sprintf("chat:%d:commands", chanID)
	val, _ := R.HKeys(key).Result()
	B.Send(m.Sender, fmt.Sprintf("commands for %s %s", chanTitle, val))
}

func registerStaticCommand(userID int, name string, text string) (error) {
	chat, _, _ := getUsersActiveChat(userID)
	key := fmt.Sprintf("chat:%d:commands", chat)
	err := R.HSet(key, name, text).Err()
	return err
}

func unregisterStaticCommand(userID int, name string) (error) {
	chanID, _, _ := getUsersActiveChat(userID)
	key := fmt.Sprintf("chat:%d:commands", chanID)
	err := R.HDel(key, name).Err()
	return err
}

func addChat(ms []*tb.Message) {
	m := ms[0]
	link := fmt.Sprintf("https://telegram.me/beru_dev_bot?startgroup=%d", m.Sender.ID)
	keys := [][]tb.InlineButton{}
	row := []tb.InlineButton{}
	button := tb.InlineButton{
		Unique: "invite_link",
		Text:   "Invite Link",
		URL:    link,
	}
	row = append(row, button)
	keys = append(keys, row)
	B.Send(m.Sender, "Click this button to invite beru to your chat", &tb.ReplyMarkup{
		InlineKeyboard: keys,
	})
}

func setUsersActiveChat(userID int, chatID int64) {
	key := fmt.Sprintf("user:%d:activeChat", userID)
	if err := R.Set(key, chatID, 0).Err(); err != nil {
		LogE.Printf("failed to set %s to %d ", key, userID)
	} else {
		LogI.Printf("%s set to %d", key, userID)
	}

}

func switchChat(ms []*tb.Message) {
	m := ms[0]
	split := strings.Split(strings.Replace(m.Text, "/switchChat ", "", 1), " ")
	chatID := split[0]
	key := fmt.Sprintf("user:%d:activeChat", m.Sender.ID)
	R.Set(key, chatID, 0)
	key = fmt.Sprintf("chat:%s:title", chatID)
	if chatName, err := R.Get(key).Result(); err == redis.Nil {
		LogE.Printf("failed to lookup title for chat %d", chatID)
	} else {
		B.Send(m.Sender, fmt.Sprintf("switched to managing chat %s", chatName))
	}
}


