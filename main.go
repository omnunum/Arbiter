package main

import (
	"time"
	"log"
	tb "gopkg.in/tucnak/telebot.v2"
	"github.com/go-redis/redis"
	"os"
	"fmt"
	"encoding/json"
)

var FunctionGroups map[string]func(m *tb.Message)
var Functions map[string]func(m *tb.Message)
var AdminFunctions map[string]func(m *tb.Message)
var CommandFunctions map[string]func(m *tb.Message)

var R *redis.Client
var B *tb.Bot

type Path struct {
	Prompts   []string `json:"prompts`
	Command   string   `json:"command"`
	Index     int      `json:"index"`
	Responses []string `json:"responses"`
}

func (p *Path) MarshalBinary() ([]byte, error) {
	return json.Marshal(&p)
}

func (p *Path) UnmarshalBinary(data []byte) error {
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}

	return nil
}

func main() {

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

	wrapPathBegin := func(p Path) func(m *tb.Message) {
		return func(m *tb.Message) {
			begin(m, p)
		}
	}
	AdminFunctions = map[string]func(m *tb.Message){
		"Add Admin": wrapPathBegin(Path{
			Prompts: []string{"Who would you like to add as an admin?"},
			Command: "/addAdmin",
		}),
		"Remove Admin": wrapPathBegin(Path{
			Prompts: []string{"Who would you like to remove as an admin?"},
			Command: "/removeAdmin",
		}),
		"View Admins": listAdmins,
	}
	CommandFunctions = map[string]func(m *tb.Message){
		"Add Command": wrapPathBegin(Path{
			Prompts: []string{"What's the name of the command?",
				"What would you like the response to be? (Markdown formatting is supported)"},
			Command: "/addCommand",
		}),
		"Remove Command": wrapPathBegin(Path{
			Prompts: []string{"What command would you like to remove?"},
			Command: "/removeCommand",
		}),
		"View Commands": listCommands,
	}
	Functions = map[string]func(m *tb.Message){
		"/addAdmin":           addAdmin,
		"/removeAdmin":        removeAdmin,
		"/viewAdmins":         listAdmins,
		"/listAdminFunctions": listAdminFunctions,
		"/switchChannel":      switchChannel,
		"/addCommand":         addCommand,
		"/removeCommand":      removeCommand,
		"/viewCommands":       listCommands,
	}
	FunctionGroups = map[string]func(m *tb.Message){
		"Manage Commands": listCommandFunctions,
		"Manage Admins":   listAdminFunctions,
		"Switch Channel": wrapPathBegin(Path{
			Prompts: []string{"Who channel would you like to switch to?"},
			Command: "/switchChannel",
		}),
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
		key := fmt.Sprintf("%d:activePath", m.Sender.ID)
		exists, err := R.Exists(key).Result()
		if err != nil {
			log.Printf("error looking up existence of active path for %s", m.Sender.ID)
			return
		}
		if exists != 0 {
			step(m)
		}
		// // check if added static command
		// if strings.HasPrefix(m.Text, "/") {
		// 	commandName := strings.Split(m.Text, " ")[0]
		// 	channel, _ := getUsersActiveChannel(client, m.Sender.ID)
		// 	key := fmt.Sprintf("%s:commands", channel)
		// 	if commandText, err := client.HGet(key, commandName).Result(); err != redis.Nil {
		//
		// 	}
		// }

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
