package main

import (
	tb "gopkg.in/tucnak/telebot.v2"
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
	Consumer ConsumerType
}

type Processor func(*tb.Message, *Prompt) *tb.Message

var Processors = map[string]Processor {

}


type Prompt struct {
	// function that receives input from previous prompt's output
	// used to create the Text and Reply
	GenerateMessage GeneratorType
	// standard message to prompt user with
	Text string
	// optional keyboard to send to user
	Reply tb.ReplyMarkup
	// input from user after prompting Text/Reply
	UserResponse *tb.Message
	// processes UserResponse before being fed to next prompt
	ProcessResponse string
}

// standard prompt when an error occurs
var ErrorPrompt = Prompt{
	Text: ErrorResponse,
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
		p := DecodePath([]byte(pathData))
		return &p
	}
}

func begin(m *tb.Message, p Path) {
	// clear out any existing active path
	key := fmt.Sprintf("user:%d:activePath", m.Sender.ID)
	err := R.Del(key).Err()
	if err != nil {
		LogE.Printf("unable to delete active path for %d %s", m.Sender.ID, err)
		return
	}
	// remove button name from path message building
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
		LogI.Print("reached the end of the path")
		if p.Consumer != "" {
			if consumer, ok := ConsumerRegistry[p.Consumer]; !ok {
				LogE.Printf("consumer not found in registry: %s", p.Consumer)
			} else {
				consumer(p.Responses)
			}
		}
		// delete the path state since it has been fully traversed
		err := R.Del(key).Err()
		if err != nil {
			LogE.Printf("unable to delete active path for user %d %s", m.Sender.ID, err)
		}
		return nil
	}
	pr := p.Prompts[p.Index]
	// if there is still a prompt in the list, send it, increment the index
	// and update the saved state data with a reset TTL
	if pr.GenerateMessage != "" {
		GeneratorRegistry[pr.GenerateMessage](m, &pr)
	}
	if m.Private() {
		B.Send(m.Sender, pr.Text, &pr.Reply)
	} else {
		B.Send(m.Chat, pr.Text, &pr.Reply)
	}
	p.Index += 1
	R.Set(key, EncodePath(p), time.Minute)
	return nil
}
