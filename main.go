package main

import (
	"github.com/go-redis/redis"
	tb "gopkg.in/tucnak/telebot.v2"
	"os"
	"time"
	"log"
	"fmt"
)

/*
STATE GUIDE
KEY FORMAT == type:instance:attribute

beru:channels <SET> : channels beru has been invited to

channel:%chatID:admins <SET> : admins for this channel that can access beru admin commands
channel:%chatID:owner <int> : super user/owner of channel, user that invited beru, can modify
	admin set
channel:%chatID:commands <MAP> : map of command names to static replies
channel:%chatID:title <string> : name of channel

user:%userID:activeChannel <string> : the channel to which the commands will affect
user:%userID:nextCommand
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
		"/switchChannel":      switchChannel,
		"/addCommand":         addCommand,
		"/removeCommand":      removeCommand,
		"/viewCommands":       listCommands,
		"/addChannel":         addChannel,
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
		// step forward if user has an active path
		key := fmt.Sprintf("user:%d:activePath", m.Sender.ID)
		exists, err := R.Exists(key).Result()
		if err != nil {
			log.Printf("error looking up existence of active path for %s", m.Sender.ID)
			return
		}
		if exists != 0 {
			step(m)
		}
		// check if added static command
		// if strings.HasPrefix(m.Text, "/") {
		// 	commandName := strings.Split(m.Text, " ")[0]
		// 	var channel string
		// 	if m.Private() {
		// 		channel, _ = getUsersActiveChannel(m.Sender.ID)
		// 	} else {
		// 		channel = m.Chat.ID
		// 	}
		// 	key := fmt.Sprintf("%s:commands", channel)
		// 	if commandText, err := R.HGet(key, commandName).Result(); err != redis.Nil {
		//
		// 	}
		// }

	})

	B.Handle(tb.OnAddedToGroup, func(m *tb.Message) {
		R.SAdd("beru:channels", m.Chat.ID)
		R.SAdd(fmt.Sprintf("channel:%d:admins", m.Chat.ID), m.Sender.ID)
		R.Set(fmt.Sprintf("channel:%d:owner", m.Chat.ID), m.Sender.ID, 0)
		R.Set(fmt.Sprintf("channel:%d:title", m.Chat.ID), m.Chat.Title, 0)
		log.Printf("beru joined channel %s (%d) invited by %s (%d)",
			m.Chat.Title, m.Chat.ID, m.Sender.Username, m.Sender.ID)
		setUsersActiveChannel(m.Sender.ID, m.Chat.ID)
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
