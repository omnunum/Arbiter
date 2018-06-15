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

var FunctionGroups = map[string]func(m *tb.Message){}
var Functions = map[string]func(m *tb.Message){}
var AdminFunctions = map[string]func(m *tb.Message){}
var CommandFunctions = map[string]func(m *tb.Message){}

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

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       15,
	})

	b, err := tb.NewBot(tb.Settings{
		Token:  os.Getenv("TELEBOT_SECRET"),
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
		return
	}

	wrapFunction := func(callable func(client *redis.Client, b *tb.Bot, m *tb.Message)) func(m *tb.Message) {
		return func(m *tb.Message) {
			callable(client, b, m)
		}
	}

	wrapPathBegin := func(p Path) func(m *tb.Message) {
		return func(m *tb.Message) {
			begin(client, b, m, p)
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
		"View Admins": wrapFunction(listAdmins),
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
		"View Commands": wrapFunction(listCommands),
	}
	Functions = map[string]func(m *tb.Message){
		"/addAdmin":           wrapFunction(addAdmin),
		"/removeAdmin":        wrapFunction(removeAdmin),
		"/viewAdmins":         wrapFunction(listAdmins),
		"/listAdminFunctions": wrapFunction(listAdminFunctions),
		"/switchChannel":      wrapFunction(switchChannel),
		"/addCommand":         wrapFunction(addCommand),
		"/removeCommand":      wrapFunction(removeCommand),
		"/viewCommands":       wrapFunction(listCommands),
	}
	FunctionGroups = map[string]func(m *tb.Message){
		"Manage Commands": wrapFunction(listCommandFunctions),
		"Manage Admins":   wrapFunction(listAdminFunctions),
		"Switch Channel": wrapPathBegin(Path{
			Prompts: []string{"Who channel would you like to switch to?"},
			Command: "/switchChannel",
		}),
	}

	for k, v := range Functions {
		b.Handle(k, v)
	}

	// Command: /start <PAYLOAD>
	b.Handle("/start", func(m *tb.Message) {
		if !m.Private() {
			return
		}
		wrapFunction(listFunctionGroups)(m)
	})

	b.Handle(tb.OnText, func(m *tb.Message) {
		// step forward if user has an active path
		key := fmt.Sprintf("%d:activePath", m.Sender.ID)
		exists, err := client.Exists(key).Result()
		if err != nil {
			log.Printf("error looking up existence of active path for %s", m.Sender.ID)
			return
		}
		if exists != 0 {
			step(client, b, m)
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

	b.Start()

	interrupt := make(chan os.Signal, 1)
	select {
	case <-interrupt:
		log.Println("interrupt")

		b.Stop()
		return
	}
}
