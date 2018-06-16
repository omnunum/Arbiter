package main

import (
	"github.com/go-redis/redis"
	tb "gopkg.in/tucnak/telebot.v2"
	"os"
	"time"
	"log"
	"fmt"
	"strings"
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

var Functions map[string]func(m *tb.Message)

func main() {
	Functions = map[string]func(m *tb.Message){
		"/addAdmin":           addAdmin,
		"/removeAdmin":        removeAdmin,
		"/viewAdmins":         listAdmins,
		"/listAdminFunctions": listAdminFunctions,
		"/switchchat":         switchChat,
		"/addCommand":         addCommand,
		"/removeCommand":      removeCommand,
		"/viewCommands":       listCommands,
		"/addchat":            addChat,
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

	for k, v := range Functions {
		B.Handle(k, v)
	}

	// Command: /start <PAYLOAD>
	B.Handle("/start", func(m *tb.Message) {
		if !m.Private() {
			return
		}
		listFunctionGroups(m)
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
				B.Send(dest, commandText)
			}
		}

	})

	B.Handle(tb.OnAddedToGroup, func(m *tb.Message) {
		// add chat to list of chats beru has been added to
		R.SAdd("beru:chats", m.Chat.ID)
		// enables a quick check that user can admin chat
		R.SAdd(fmt.Sprintf("user:%d:chats", m.Sender.ID), m.Chat.ID)
		// add user to chats admin list
		R.SAdd(fmt.Sprintf("chat:%d:admins", m.Chat.ID), m.Sender.ID)
		// since this is the inviter, add this user as the owner of the chat
		R.Set(fmt.Sprintf("chat:%d:owner", m.Chat.ID), m.Sender.ID, 0)
		// save the title so we can display this lto humans
		R.Set(fmt.Sprintf("chat:%d:title", m.Chat.ID), m.Chat.Title, 0)
		log.Printf("beru joined chat %s (%d) invited by %s (%d)",
			m.Chat.Title, m.Chat.ID, m.Sender.Username, m.Sender.ID)
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
