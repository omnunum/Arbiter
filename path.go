package main

import (
	tb "gopkg.in/tucnak/telebot.v2"
	"fmt"
	"time"
)

type Path struct {
	// chain of text and keyboards to prompt user with
	Prompts []Prompt
	// tracks the progress through the path
	Index int
	// stores the message objects that users respond to the
	// prompts with
	Responses []*tb.Message
	// reference to function registry value that consumes
	// the list of responses after all prompts have been sent to user
	Consumer ConsumerType
	// path is only available to chat owner
	OwnerOnly bool
}

type Processor func(*tb.Message, *Prompt) *tb.Message

var Processors = map[string]Processor{

}

type Prompt struct {
	// function that receives input from previous prompt's output
	// used to create the Text and Reply
	GenerateMessage GeneratorType
	// standard message to prompt user with
	Text string
	// optional keyboard to send to user
	Reply tb.ReplyMarkup
	// optional text to send along as a keyboard
	Buttons [][]string
	// input from user after prompting Text/Reply
	UserResponse *tb.Message
	// processes UserResponse before being fed to next prompt
	ProcessResponse string
}

// standard prompt when an error occurs
var ErrorPrompt = Prompt{
	Text: ErrorResponse,
}

func wrapPathBegin(p Path) func(m *tb.Message) {
	return func(m *tb.Message) {
		begin(m, p)
	}
}

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
		B.Send(m.Sender, ErrorResponse)
		return
	}
	chatID, chanTitle, err := getUsersActiveChat(m.Sender.ID)
	if err != nil {
		LogE.Printf("couldn't lookup active chat for user: %s", m.Sender.ID)
		B.Send(m.Sender, ErrorResponse)
		return
	}
	access, err := userHasAdminManagementAccess(m.Sender.ID, chatID)
	if err != nil {
		LogE.Printf("couldn't lookup admin access for user: %s", m.Sender.ID)
		B.Send(m.Sender, ErrorResponse)
		return
	}
	if !access && p.OwnerOnly {
		msg := fmt.Sprintf("You don't have admin management access for %s.", chanTitle)
		B.Send(m.Sender, msg)
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
	} else if len(pr.Buttons) > 0{
		rows := [][]tb.ReplyButton{}
		for _, br := range(pr.Buttons) {
			row := []tb.ReplyButton{}
			for _, b := range(br) {
				row = append(row, tb.ReplyButton{
					Text: b,
				})
			}
			rows = append(rows, row)
		}
		pr.Reply = tb.ReplyMarkup{
			ReplyKeyboard:       rows,
			ResizeReplyKeyboard: true,
			OneTimeKeyboard:     true,
		}
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
