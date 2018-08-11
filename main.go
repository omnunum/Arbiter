package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"
	"time"
	"github.com/go-redis/redis"
	tb "gopkg.in/tucnak/telebot.v2"
	"strconv"
)

/*
STATE GUIDE
KEY FORMAT == type:instance:attribute

beru:chats <SET> : chats beru has been invited to

chat:%chatID:admins <SET> : admins for this chat that can access beru admin commands
chat:%chatID:owner <int> : super user/owner of chat, user that invited beru, can modify
	admin set
chat:%chatID:commands <MAP> : map of command names to static replies
chat:%chatID:title <string> : name of chat
chat:%chatID:usersJoinedCount <int> : number of users joined since beru started tracking
chat:%chatID:usersJoinedLimit <int> : number of users joined before beru posts welcome message
chat:%chatID:usersJoinedMessage <string> : welcome message to post

user:%userID:activechat <string> : the chat to which the commands will affect
user:%userID:activePath <Path> : the user dialogue Path that has been started,
	but not fully traversed
user:%userID:chats <SET> : quick lookup to see what chats user is admin/owner of
user:%user:info <tb.User> : user object for looking up user details
*/

var R *redis.Client
var B *tb.Bot

const ErrorResponse string = "Something went wrong and I wasn't able to fulfill that request"

var helpGuide = `
Nice you meet you, my name is Beru!

I can help you create and manage Telegram chats.

You can control me by sending these commands:

*Chat Owner Only*
/addadmin - allows another user to change the chat rules
/removeadmin - removes a users ability to change chat rules
/viewadmins - displays list of users with admin privileges

*Beru Level Functionality*
/switchchat - changes which chat beru is managing when a user is an owner/admin of multiple chats
/addchat - shortcut to invite link to add beru to your chat
/removechat - choose between currently managed chats to remove

*Custom Chat Commands*
/addcommand - adds a custom command and response 
/removecommand - removes a custom command
/viewcommands - prints a list of custom commands

*Chat Features*
/setwelcome - greets every # users with a welcome message on chat join
/togglejoinmsg - toggles deletion of the notification posted when users join (supergroups only)
/addwhitelistedbot - adds a bot (by username) to be allowed to join a chat
/removewhitelistedbot - removes a bots ability to join a chat`

func main() {
	gob.Register(Path{})
	gob.Register(Prompt{})
	gob.Register(tb.User{})
	gob.Register(tb.Chat{})

	R = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       15,
	})
	var err error
	B, err = tb.NewBot(tb.Settings{
		Token:  os.Getenv("TELEBOT_SECRET"),
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		panic(err)
		return
	}

	for k, v := range BuiltinCommandRegistry {
		B.Handle(k, v)
	}

	// Command: /start <PAYLOAD>
	B.Handle("/start", func(m *tb.Message) {
		if !m.Private() {
			return
		}
		// if the user isnt an admin of any chats
		key := fmt.Sprintf("user:%d:chats", m.Sender.ID)
		if nChats := R.SCard(key).Val(); nChats == 0 {
			B.Send(m.Sender, "I need to be invited to a chat before I can be useful")
			addChat([]*tb.Message{m})
		} else {
			listFunctionGroups(m)
		}
	})

	// Command: /start <PAYLOAD>
	B.Handle("/help", func(m *tb.Message) {
		if !m.Private() {
			return
		}
		B.Send(m.Sender, helpGuide, tb.ParseMode(tb.ModeMarkdown))
	})

	B.Handle(tb.OnText, func(m *tb.Message) {
		if p := getUsersActivePath(m.Sender.ID); p != nil {
			step(m, p)
		}
		// check if command
		if strings.HasPrefix(m.Text, "/") {
			commandName := strings.Split(m.Text, " ")[0]
			var chat int
			var dest tb.Recipient
			// if chatting with beru, respond to user, else chat
			if m.Private() {
				chat, _, _ = getUsersActiveChat(m.Sender.ID)
				dest = m.Sender
			} else {
				chat = int(m.Chat.ID)
				dest = m.Chat
			}
			key := fmt.Sprintf("chat:%d:commands", chat)
			if commandText, err := R.HGet(key, commandName).Result(); err != redis.Nil {
				t, _ := template.New("command").Parse(commandText)
				by := bytes.Buffer{}
				if err := t.Execute(&by, m); err != nil {
					LogE.Printf("failed to render template for command %s in chat %d", commandName, chat)
					B.Send(dest, ErrorResponse)
				} else {
					B.Send(dest, by.String())
				}
			}
		}

	})

	B.Handle("/admins", func(m *tb.Message) {
		if m.Private() {
			return
		}
		if members, err := B.AdminsOf(m.Chat); err != nil {
			LogE.Printf("error fetching admins for chat %s", m.Chat.ID)
			B.Send(m.Sender, ErrorResponse)
		} else {
			var usernameList = []string{}
			for _, u := range members {
				usernameList = append(usernameList, "@"+u.User.Username)
			}

			B.Send(m.Chat, fmt.Sprintf("Hey %s %s, the admins for this channel are: %s",
				m.Sender.FirstName, m.Sender.LastName, strings.Join(usernameList, ", ")))
		}
	})

	B.Handle("/price", func(m *tb.Message) {
		if m.Private() {
			return
		}
		key := fmt.Sprintf("chat:%d:price", m.Chat.ID)
		slug, err := R.HGet(key, "slug").Result()
		conversion, err := R.HGet(key, "conversion").Result()
		msgFormat, err := R.HGet(key, "msgFormat").Result()
		if err != nil {
			B.Send(m.Chat, ErrorResponse)
			LogE.Print(err)
			return
		}

		token := getTokenInfo(slug)
		price, converted, pct_price, pct_conversion := getTokenPrice(token.ID, conversion)
		// these are all the possible tags a user can use in the message formatter
		replacer := strings.NewReplacer(
			"{{price}}", strconv.FormatFloat(price, 'f', 5, 64),
			"{{ticker}}", token.Symbol,
			"{{name}}", token.Name,
			"{{slug}}", slug,
			"{{conversion}}", strconv.FormatFloat(converted, 'f', 8, 64),
			"{{price_pct_change}}", fmt.Sprintf("%+.1f%%", pct_price),
			"{{conversion_pct_change}}", fmt.Sprintf("%+.1f%%", pct_conversion),
		)
		replaced := replacer.Replace(msgFormat)
		B.Send(m.Chat, replaced)
	})

	B.Handle(tb.OnUserJoined, func(m *tb.Message) {
		// kick bot if not whitelisted
		k := fmt.Sprintf("chat:%d:botWhitelist", m.Chat.ID)
		for _, u := range m.UsersJoined {
			// we only need to lookup users that are bots
			if !strings.HasSuffix(strings.ToLower(u.Username), "bot") {
				continue
			}
			// for all bots, ban if not member or if whitelist was never set up
			isMember, err := R.SIsMember(k, u.Username).Result()
			if err != nil || !isMember {
				B.Ban(m.Chat, &tb.ChatMember{User: &u, RestrictedUntil: tb.Forever()})
				B.Send(m.Chat, "fuck ur bot")
			}
		}

		// post welcome message if available
		countKey := fmt.Sprintf("chat:%d:usersJoinedCount", m.Chat.ID)
		limitKey := fmt.Sprintf("chat:%d:usersJoinedLimit", m.Chat.ID)
		messageKey := fmt.Sprintf("chat:%d:usersJoinedMessage", m.Chat.ID)
		usersJoined, _ := R.Incr(countKey).Result()
		usersLimit, _ := R.Get(limitKey).Int64()
		if usersJoined%usersLimit == 0 {
			joinedMsg, _ := R.Get(messageKey).Result()
			fmtMsg := strings.Replace(joinedMsg, "$username", m.Sender.Username, -1)
			B.Send(m.Chat, fmtMsg)
		}
		// delete join notification if setting is set to on
		deleteKey := fmt.Sprintf("chat:%d:deleteJoinNotification", m.Chat.ID)
		delete, err := R.Get(deleteKey).Int64()
		if err == nil && delete != 0 {
			B.Delete(m)
		}
	})

	B.Handle(tb.OnAddedToGroup, func(m *tb.Message) {
		// add chat to list of chats beru has been added to
		R.SAdd("beru:chats", m.Chat.ID)
		// enables a quick check that user can admin chat
		R.SAdd(fmt.Sprintf("user:%d:chats", m.Sender.ID), m.Chat.ID)
		// save the full user info if we need it later
		R.Set(fmt.Sprintf("user:%d:info", m.Sender.ID), EncodeUser(m.Sender), 0)
		// // add user to active chats admin list
		R.SAdd(fmt.Sprintf("chat:%d:activeAdmins", m.Chat.ID), m.Sender.ID)
		// since this is the inviter, add this user as the owner of the chat
		R.Set(fmt.Sprintf("chat:%d:owner", m.Chat.ID), m.Sender.ID, 0)
		// save the chat title so we can display it to the user
		R.Set(fmt.Sprintf("chat:%d:title", m.Chat.ID), m.Chat.Title, 0)
		// save the full chat info if we need it later
		R.Set(fmt.Sprintf("chat:%d:info", m.Chat.ID), EncodeChat(m.Chat), 0)
		// set user join notification deletion to off
		R.Set(fmt.Sprintf("chat:%d:deleteJoinNotification", m.Chat.ID), 0, 0)
		// add all chat admins to list so we can prompt user with potential
		// options when adding and removing admins
		if members, err := B.AdminsOf(m.Chat); err != nil {
			LogE.Printf("error fetching admins for chat %s", m.Chat.ID)
			B.Send(m.Sender, ErrorResponse)
		} else {
			for _, u := range members {
				R.Set(fmt.Sprintf("user:%d:info", u.User.ID),
					EncodeUser(u.User), 0)
				R.SAdd(fmt.Sprintf("chat:%d:admins", m.Chat.ID), u.User.ID)
			}
		}
		LogI.Printf("beru joined chat %s (%d) invited by %s (%d)",
			m.Chat.Title, m.Chat.ID, m.Sender.Username, m.Sender.ID)
		B.Send(m.Sender, fmt.Sprintf("beru joined chat %s", m.Chat.Title))
		setUsersActiveChat(m.Sender.ID, m.Chat.ID)

	})

	B.Start()

	interrupt := make(chan os.Signal, 1)
	select {
	case <-interrupt:
		log.Println("interrupt")

		B.Stop()
		return
	}
}
