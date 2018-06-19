package main

import (
	"github.com/go-redis/redis"
	tb "gopkg.in/tucnak/telebot.v2"
	"os"
	"time"
	"log"
	"fmt"
	"strings"
	"encoding/gob"
	"text/template"
	bytes2 "bytes"
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

user:%userID:activechat <string> : the chat to which the commands will affect
user:%userID:activePath <Path> : the user dialogue Path that has been started,
	but not fully traversed
user:%userID:chats <SET> : quick lookup to see what chats user is admin/owner of
*/

var R *redis.Client
var B *tb.Bot

var Consumers map[string]func([]*tb.Message)

const ErrorResponse string = "Something went wrong and I wasn't able to fulfill that request"

var helpGuide = `
Nice you meet you, my name is Beru!

I can help you create and manage Telegram chats.

You can control me by sending these commands:

*Chat Owner Only*
/addadmin - allows another user to change the chat rules
/removeadmin - removes a users ability to change chat rules
/viewadmins - displays list of users with admin privileges

*Chat Level Functionality*
/switchchat - changes which chat beru is managing when a user is an owner/admin of multiple chats
/addchat - shortcut to invite link to add beru to your chat

*Custom Commands*
/addcommand - adds a custom command and response 
/removecommand - removes a custom command
/viewcommands - prints a list of custom commands
`

func main() {
	gob.Register(Path{})
	gob.Register(Prompt{})
	Consumers = map[string]func([]*tb.Message){
		"/addadmin":      addAdmin,
		"/removeadmin":   removeAdmin,
		"/viewadmins":    viewAdmins,
		"/addchat":       addChat,
		"/switchchat":    switchChat,
		"/addcommand":    addCommand,
		"/removecommand": removeCommand,
		"/viewcommands":  viewCommands,
	}

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
		log.Fatal(err)
		return
	}

	// for k, v := range Consumers {
	// 	B.Handle(k, v)
	// }

	// Command: /start <PAYLOAD>
	B.Handle("/start", func(m *tb.Message) {
		if !m.Private() {
			return
		}
		// if there isn't an active chat for this user
		key := fmt.Sprintf("user:%d:activeChat", m.Sender.ID)
		if exists := R.Exists(key).Val(); exists == 0 {
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
		// check if added static command
		if strings.HasPrefix(m.Text, "/") {
			commandName := strings.Split(m.Text, " ")[0]
			var chat int
			var dest tb.Recipient
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
				by := bytes2.Buffer{}
				if err := t.Execute(&by, m); err != nil {
					B.Send(dest, ErrorResponse)
				} else {
					B.Send(dest, by.String())
				}
			}
		}

	})

	B.Handle(tb.OnAddedToGroup, func(m *tb.Message) {
		// add chat to list of chats beru has been added to
		R.SAdd("beru:chats", m.Chat.ID)
		// enables a quick check that user can admin chat
		R.SAdd(fmt.Sprintf("user:%d:chats", m.Sender.ID), m.Chat.ID)
		// save the full user info if we need it later
		R.Set(fmt.Sprintf("user:%d:info", m.Sender.ID), EncodeUser(m.Sender), 0)
		// add user to chats admin list
		R.SAdd(fmt.Sprintf("chat:%d:admins", m.Chat.ID), m.Sender.ID)
		// since this is the inviter, add this user as the owner of the chat
		R.Set(fmt.Sprintf("chat:%d:owner", m.Chat.ID), m.Sender.ID, 0)
		// save the chat title so we can display it to the user
		R.Set(fmt.Sprintf("chat:%d:title", m.Chat.ID), m.Chat.Title, 0)
		// save the full chat info if we need it later
		R.Set(fmt.Sprintf("chat:%d:info", m.Chat.ID), EncodeChat(m.Chat), 0)

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
