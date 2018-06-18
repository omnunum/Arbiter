package main

import (
	tb "gopkg.in/tucnak/telebot.v2"
	"encoding/gob"
	"bytes"
	"log"
	"fmt"
	"time"
)

type Path struct {
	// chain of text and keyboards to promp user with
	Prompts []Prompt
	// tracks the progress through the path
	Index int
	// stores the message objects that users respond to the
	// prompts with
	Responses []*tb.Message
	// reference to function registry value that consumes
	// the list of responses after all prompts have been sent to user
	Consumer string
}
type Processor func(*tb.Message) *tb.Message

var Processors = map[string]Processor {

}

type Generator func(interface{})

var Generators = map[string]Generator {

}

type Prompt struct {
	// function that receives input from previous prompt's output
	// used to create the Text and Reply
	GenerateMessage string
	// standard message to prompt user with
	Text string
	// optional keyboard to send to user
	Reply tb.ReplyMarkup
	// input from user after prompting Text/Reply
	UserResponse *tb.Message
	// processes UserResponse before being fed to next prompt
	ProcessResponse string
}

func Encode(p Path) []byte {
	var by bytes.Buffer
	enc := gob.NewEncoder(&by)
	if err := enc.Encode(p); err != nil {
		log.Printf("could not gob encode path with prompt %s", p.Prompts[0])
		panic(err)
	}
	data := by.Bytes()
	return data
}

func Decode(data []byte) Path {
	var by bytes.Buffer
	by.Write(data)
	dec := gob.NewDecoder(&by)
	p := Path{}
	if err := dec.Decode(&p); err != nil {
		log.Printf("Unable to decode data into the new Path struct due to %s", err)
		panic(err)
	}
	return p
}

func wrapPathBegin(p Path) func(ms *tb.Message) {
	return func(m *tb.Message) {
		begin(m, p)
	}
}

// var Paths = map[string]Path{
// 	"addAllAdminsOnChatInvitation": Path{
// 		Prompts: []string{"Would you like to add all the admins of this channel?"},
// 		Consumer: "/addAdmin",
// 	},
// }

func getUsersActivePath(userID int) *Path {
	key := fmt.Sprintf("user:%d:activePath", userID)
	pathData, err := R.Get(key).Result()
	if err != nil {
		return nil
	} else {
		// decode path data into native struct
		p := Decode([]byte(pathData))
		return &p
	}
}

func begin(m *tb.Message, p Path) {
	// clear out any existing active path
	key := fmt.Sprintf("user:%d:activePath", m.Sender.ID)
	err := R.Del(key).Err()
	if err != nil {
		log.Printf("unable to delete active path for %d %s", m.Sender.ID, err)
		return
	}
	// remove button name from path messasge building
	m.Text = ""
	// take the first step
	step(m, &p)
}

func step(m *tb.Message, p *Path) error {
	key := fmt.Sprintf("user:%d:activePath", m.Sender.ID)
	// if the incoming message has text append it to the list of responses.
	if m.Text != "" {
		p.Responses = append(p.Responses, m)
	}
	// if all of the prompts have been sent to the user call the function
	// at the end of the path, and pass in the responses joined by a semicolon
	if p.Index == len(p.Prompts) {
		log.Print("reached the end of the path")
		if consumer, ok := Consumers[p.Consumer]; !ok {
			log.Printf("consumer not found in registry: %s", p.Consumer)
		} else {
			consumer(p.Responses)
		}
		// delete the path state since it has been fully traversed
		_, err := R.Del(key).Result()
		if err != nil {
			log.Printf("unable to delete active path for user %d %s", m.Sender.ID, err)
		}
		return nil
	}
	pr := p.Prompts[p.Index]
	// if there is still a prompt in the list, send it, increment the index
	// and update the saved state data with a reset TTL
	if pr.GenerateMessage != "" {
		Generators[pr.GenerateMessage](m)
	}
	if m.Private() {
		B.Send(m.Sender, pr.Text, &pr.Reply)
	} else {
		B.Send(m.Chat, pr.Text, &pr.Reply)
	}
	p.Index += 1
	R.Set(key, Encode(*p), time.Minute)
	return nil
}
