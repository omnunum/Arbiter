package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-redis/redis"
	"github.com/pkg/errors"
	tb "gopkg.in/tucnak/telebot.v2"
)

// needs to be string constant so we can encode with gob
// but still refer to functions, since functions can't be
// encoded with gob
type ConsumerType string

const (
	CAddAdmin              ConsumerType = "/addadmin"
	CViewAdmins            ConsumerType = "/viewadmins"
	CRemoveAdmin           ConsumerType = "/removeadmin"
	CRemoveChat            ConsumerType = "/removechat"
	CAddChat               ConsumerType = "/addchat"
	CSwitchChat            ConsumerType = "/switchchat"
	CAddCommand            ConsumerType = "/addcommand"
	CRemoveCommand         ConsumerType = "/removecommand"
	CViewCommands          ConsumerType = "/viewcommands"
	CSetWelcome            ConsumerType = "/setwelcome"
	CToggleJoinMessage     ConsumerType = "/togglejoinmsg"
	CRemoveWhitelistedBot  ConsumerType = "/removewhitelistedbot"
	CAddWhitelistedBot     ConsumerType = "/addwhitelistedbot"
	CSetPriceCommand       ConsumerType = "/setpricecommand"
	CSetNewUserRestriction ConsumerType = "/setnewusermediarestriction"
)

type Consumer func([]*tb.Message) error

var ConsumerRegistry = map[ConsumerType]Consumer{
	CAddAdmin:              addAdmin,
	CRemoveAdmin:           removeAdmin,
	CViewAdmins:            viewAdmins,
	CAddChat:               addChat,
	CSwitchChat:            switchChat,
	CRemoveChat:            removeChat,
	CAddCommand:            addCommand,
	CRemoveCommand:         removeCommand,
	CViewCommands:          viewCommands,
	CSetWelcome:            setWelcome,
	CToggleJoinMessage:     toggleJoinMessage,
	CRemoveWhitelistedBot:  removeWhitelistedBot,
	CAddWhitelistedBot:     addWhitelistedBot,
	CSetPriceCommand:       setPriceCommand,
	CSetNewUserRestriction: setNewUserMediaRestriction,
}

// consts for switching basic consumer behavior
const (
	cRem int = iota
	cGet
	cSet
)

func getReplyKeyboardForCommands(commands []FunctionButton) [][]tb.ReplyButton {
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

func getInlineKeyboardForCommands(commands []FunctionButton) [][]tb.InlineButton {
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

func getInlineButtonForMessages(buttonTexts []string) [][]tb.InlineButton {
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

// utility used by the set, get, and remove admin consumers
func accessAdmins(userID int, operation int, adminID ...string) (msg string, err error) {
	chatID, chanTitle, err := getUsersActiveChat(userID)
	if err != nil {
		err = errors.Wrapf(err, "couldn't get active chat")
		return "", err
	}
	activeKey := fmt.Sprintf("chat:%d:activeAdmins", chatID)
	switch operation {
	case cGet:
		var val []string
		val, err = R.SMembers(activeKey).Result()
		usernames := []string{}
		for _, userID := range val {
			userIDInt, _ := strconv.Atoi(userID)
			username, _ := getUserName(userIDInt)
			usernames = append(usernames, fmt.Sprintf("@%s", username))
		}
		msg = fmt.Sprintf("admins for chat %s are %s", chanTitle, usernames)
	case cRem:
		err = R.SRem(activeKey, adminID[0]).Err()
		userChatsKey := fmt.Sprintf("user:%s:chats", adminID[0])
		err = R.SRem(userChatsKey, chatID).Err()
		msg = fmt.Sprintf("admin removed: %s", adminID[0])
	case cSet:
		err = R.SAdd(activeKey, adminID[0]).Err()
		userChatsKey := fmt.Sprintf("user:%s:chats", adminID[0])
		err = R.SAdd(userChatsKey, chatID).Err()
		userActiveChatKey := fmt.Sprintf("user:%d:activeChat", adminID[0])
		if yes, _ := R.Exists(userActiveChatKey).Result(); yes == 0 {
			R.Set(userActiveChatKey, chatID, 0)
		}
		msg = fmt.Sprintf("admin added: %s", adminID[0])
	}
	err = errors.Wrap(err, "")
	return
}

func addAdmin(ms []*tb.Message) (err error) {
	m := ms[0]
	split := strings.Split(strings.Replace(m.Text, "/addAdmin ", "", 1), " ")
	adminName := split[0]
	if msg, err := accessAdmins(m.Sender.ID, cSet, adminName); err != nil {
		B.Send(m.Sender, ErrorResponse)
		return errors.Wrapf(err, "couldn't add admin: %s")
	} else {
		B.Send(m.Sender, msg)
	}
	return
}

func removeAdmin(ms []*tb.Message) (err error) {
	m := ms[0]
	split := strings.Split(strings.Replace(m.Text, "/removeAdmin ", "", 1), " ")
	adminName := split[0]
	if msg, err := accessAdmins(m.Sender.ID, cRem, adminName); err != nil {
		B.Send(m.Sender, ErrorResponse)
		return errors.Wrapf(err, "couldn't remove admin: %s", adminName)
	} else {
		B.Send(m.Sender, msg)
	}
	return
}

func viewAdmins(ms []*tb.Message) (err error) {
	m := ms[0]
	if msg, err := accessAdmins(m.Sender.ID, cGet); err != nil {
		B.Send(m.Sender, ErrorResponse)
		return errors.Wrap(err, "couldn't get admins")
	} else {
		B.Send(m.Sender, msg)
	}
	return
}

func addCommand(ms []*tb.Message) (err error) {
	sender := ms[0].Sender
	commandName := ms[0].Text
	if !strings.HasPrefix(commandName, "/") {
		commandName = "/" + commandName
	}
	commandText := ms[1].Text
	LogI.Printf("entered with msgs [%s %s]", commandName, commandText)
	if len(ms) == 1 {
		B.Send(sender, fmt.Sprint(
			"you need to specify a command and response to add, such as /addCommand commandName;response text"))
	}
	if err = registerStaticCommand(sender.ID, commandName, commandText); err != nil {
		msg := fmt.Sprintf("error while trying to add command %s", commandName)
		B.Send(sender, msg)
		return errors.Wrapf(err, msg)
	}
	B.Send(sender, fmt.Sprintf("added/updated command %s", commandName))
	return
}

func removeCommand(ms []*tb.Message) (err error) {
	m := ms[0]
	LogI.Printf("entered with msg %s", m.Text)
	commandName := strings.Replace(m.Text, "/removeCommand ", "", 1)
	if !strings.HasPrefix(commandName, "/") {
		commandName = "/" + commandName
	}
	if err = unregisterStaticCommand(m.Sender.ID, commandName); err != nil {
		msg := fmt.Sprintf("error while trying to remove command %s", commandName)
		B.Send(m.Sender, msg)
		return errors.Wrapf(err, msg)
	}
	B.Send(m.Sender, fmt.Sprintf("removed command %s", commandName))
	return
}

func viewCommands(ms []*tb.Message) (err error) {
	m := ms[0]
	chanID, chanTitle, _ := getUsersActiveChat(m.Sender.ID)
	key := fmt.Sprintf("chat:%d:commands", chanID)
	val, err := R.HKeys(key).Result()
	if err != nil {
		return errors.Wrapf(err, "could not access keys of %s", key)
	}
	B.Send(m.Sender, fmt.Sprintf("commands for %s %s", chanTitle, val))
	return
}

func registerStaticCommand(userID int, name string, text string) (err error) {
	chat, _, _ := getUsersActiveChat(userID)
	key := fmt.Sprintf("chat:%d:commands", chat)
	return R.HSet(key, name, text).Err()
}

func unregisterStaticCommand(userID int, name string) (err error) {
	chanID, _, _ := getUsersActiveChat(userID)
	key := fmt.Sprintf("chat:%d:commands", chanID)
	return R.HDel(key, name).Err()
}

func addChat(ms []*tb.Message) (err error) {
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
	return
}

func setUsersActiveChat(userID int, chatID int64) (err error) {
	key := fmt.Sprintf("user:%d:activeChat", userID)
	if err := R.Set(key, chatID, 0).Err(); err != nil {
		return errors.Wrapf(err, "failed to set %s to %d ", key, userID)
	} else {
		LogI.Printf("%s set to %d", key, userID)
	}
	return
}

func switchChat(ms []*tb.Message) (err error) {
	m := ms[0]
	split := strings.Split(strings.Replace(m.Text, "/switchChat ", "", 1), " ")
	chatID := split[0]
	key := fmt.Sprintf("user:%d:activeChat", m.Sender.ID)
	R.Set(key, chatID, 0)
	key = fmt.Sprintf("chat:%s:title", chatID)
	if chatName, err := R.Get(key).Result(); err == redis.Nil {
		return errors.Wrapf(err, "failed to lookup title for chat %d", chatID)
	} else {
		B.Send(m.Sender, fmt.Sprintf("switched to managing chat %s", chatName))
	}
	return
}

func removeChat(ms []*tb.Message) (err error) {
	m := ms[0]
	split := strings.Split(strings.Replace(m.Text, "/removeChat ", "", 1), " ")
	chatID := split[0]
	activeKey := fmt.Sprintf("user:%d:activeChat", m.Sender.ID)
	activeChat, err := R.Get(activeKey).Result()
	titleKey := fmt.Sprintf("chat:%s:title", chatID)
	var chatName = ""
	if chatName, err = R.Get(titleKey).Result(); err == redis.Nil {
		return errors.Wrapf(err, "failed to lookup title for chat %d", chatID)
	}
	if activeChat == chatID {
		R.Set(fmt.Sprintf("user:%s:activeChat", m.Sender.ID), 0, 0)
	}

	R.Del(fmt.Sprintf("chat:%s:owner", chatID))
	R.Del(fmt.Sprintf("chat:%s:title", chatID))
	R.Del(fmt.Sprintf("chat:%s:activeAdmins", chatID))
	adminsKey := fmt.Sprintf("chat:%s:admins", chatID)
	if allAdmins, err := R.SMembers(adminsKey).Result(); err != redis.Nil {
		for _, admin := range allAdmins {
			// remove access to the chat getting deleted
			R.SRem(fmt.Sprintf("user:%s:chats", admin), chatID)
			// convert admins id to int
			userID, _ := strconv.Atoi(admin)
			// get active chat of admin we're about to remove access
			adminsActiveChat, _, _ := getUsersActiveChat(userID)
			// if the admin in question has the chat about to be removed
			// as their active chat
			if chatID == strconv.Itoa(adminsActiveChat) {
				// get other chats available to admin
				chats, err := R.SMembers(fmt.Sprintf("user:%s:chats", admin)).Result()
				if err != nil && len(chats) > 0 {
					R.Set(fmt.Sprintf("user:%s:activeChat", admin), chats[0], 0)
				} else {
					R.Set(fmt.Sprintf("user:%s:activeChat", admin), 0, 0)
				}

			} else {
				R.Set(fmt.Sprintf("user:%s:activeChat", admin), 0, 0)
			}
		}
	}
	id, err := strconv.Atoi(chatID)
	B.Leave(&tb.Chat{ID: int64(id)})
	R.Del(fmt.Sprintf("chat:%s:info", chatID))
	R.Del(fmt.Sprintf("chat:%s:admins", chatID))
	B.Send(m.Sender, fmt.Sprintf("removed beru management of chat %s", chatName))
	return
}

func setWelcome(ms []*tb.Message) (err error) {
	chatID, _, _ := getUsersActiveChat(ms[0].Sender.ID)
	message, count := ms[0].Text, ms[1].Text
	countInt, err := strconv.Atoi(count)
	if err != nil {
		B.Send(ms[0].Sender, fmt.Sprintf("User count needs to be a number, \"%s\" is not a number", count))
		return
	}
	limitKey := fmt.Sprintf("chat:%d:usersJoinedLimit", chatID)
	messageKey := fmt.Sprintf("chat:%d:usersJoinedMessage", chatID)
	err = R.Set(messageKey, message, 0).Err()
	err = R.Set(limitKey, countInt, 0).Err()
	return
}

func toggleJoinMessage(ms []*tb.Message) (err error) {
	chatID, _, _ := getUsersActiveChat(ms[0].Sender.ID)

	deleteMsgKey := fmt.Sprintf("chat:%d:deleteJoinNotification", chatID)
	delete, err := R.Get(deleteMsgKey).Int64()
	// if this hasn't been set before we're going to set it to 0 so when we
	// toggle it the feature ends up enabled
	if err != nil {
		delete = 1
	}
	delete = delete ^ 1
	R.Set(fmt.Sprintf("chat:%d:deleteJoinNotification", chatID), delete, 0)
	state := "on"
	// if delete is turned off, tell user notifications are on
	if delete != 0 {
		state = "off"
	}
	B.Send(ms[0].Sender, "User join notifications have been turned "+state)
	return
}

func addWhitelistedBot(ms []*tb.Message) (err error) {
	chatID, _, err := getUsersActiveChat(ms[0].Sender.ID)
	botName := ms[0].Text
	if !strings.HasSuffix(strings.ToLower(botName), "bot") {
		B.Send(ms[0].Sender, botName+" is not a bot")
		return
	}
	err = R.SAdd(fmt.Sprintf("chat:%d:botWhitelist", chatID), botName).Err()
	B.Send(ms[0].Sender, botName+" has been added to the whitelist")
	B.Unban(&tb.Chat{ID: int64(chatID)}, &tb.User{Username: botName})
	return
}

func removeWhitelistedBot(ms []*tb.Message) (err error) {
	chatID, _, err := getUsersActiveChat(ms[0].Sender.ID)
	botName := ms[0].Text
	err = R.SRem(fmt.Sprintf("chat:%d:botWhitelist", chatID), botName, 0).Err()
	B.Send(ms[0].Sender, botName+" has been removed from the whitelist")
	return
}

func setPriceCommand(ms []*tb.Message) (err error) {
	chatID, _, err := getUsersActiveChat(ms[0].Sender.ID)
	slug, conversion, msgFormat := ms[0].Text, ms[1].Text, ms[2].Text

	priceKey := fmt.Sprintf("chat:%d:price", chatID)
	R.HSet(priceKey, "slug", slug)
	R.HSet(priceKey, "conversion", conversion)
	R.HSet(priceKey, "msgFormat", msgFormat)

	B.Send(ms[0].Sender, "/price command has been enabled and set to report  "+slug)
	return
}

// chat owners can set an optional flag that prevents new users
// from posting URL links, and media to the chat until the
// restriction time has elapsed and the flag is removed from redis
func setNewUserMediaRestriction(ms []*tb.Message) (err error) {
	chatID, _, err := getUsersActiveChat(ms[0].Sender.ID)
	key := fmt.Sprintf("chat:%d:userRestrictionTime", chatID)
	sender := ms[0].Sender
	timescale := ms[0].Text
	timeUnits, err := strconv.ParseInt(ms[1].Text, 10, 0)
	if err != nil {
		LogE.Print(err)
		B.Send(sender, ErrorResponse + "\nyou need to specify a number as the time units")
		return
	}
	time := 0
	switch timescale{
	case "Weeks":
		time = 7 * 24 * 60 * int(timeUnits)
	case "Days":
		time = 24 * 60 * int(timeUnits)
	case "Hours":
		time = 60 * int(timeUnits)
	case "Minutes":
		time = int(timeUnits)
	default:
		B.Send(sender, ErrorResponse + "\nYou need to specify a valid timescale (Days, Weeks, Hours, Minutes)")
		return
	}
	R.Set(key, time * 60, 0)
	B.Send(sender, fmt.Sprintf("New users restricted from posting media for %d %s", timeUnits, timescale))
	return
}
