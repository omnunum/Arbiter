package main

import (
	tb "gopkg.in/tucnak/telebot.v2"
	"fmt"
	"strconv"
)

// needs to be string constant so we can encode with gob
// but still refer to functions, since functions can't be
// encoded with gob
type GeneratorType string

const (
	GSwitchChat GeneratorType = "SwitchChatGenerator"
)

type Generator func(*tb.Message, *Prompt)

var GeneratorRegistry = map[GeneratorType]Generator{
	GSwitchChat: SwitchChatGenerator,
}

func SwitchChatGenerator(m *tb.Message, pr *Prompt) {
	userID := m.Sender.ID
	// grab chat ids associated with user
	k := fmt.Sprintf("user:%d:chats", userID)
	chatIDs, err := R.SMembers(k).Result()
	if err != nil {
		LogE.Printf("couldn't get chat IDs for %s: %s", userID, err)
		pr = &ErrorPrompt
		return
	}
	// build keyboard for chat selection
	keys := [][]tb.ReplyButton{}
	row := []tb.ReplyButton{}
	for _, c := range chatIDs {
		// convert chatID strings to ints
		if id, err := strconv.Atoi(c); err != nil {
			LogE.Printf("can't convert userID %s into int: %s", c, err)
			pr = &ErrorPrompt
			return
		} else {
			// create a button for each chat
			chatTitle, _ := getChatTitle(id)
			wrappedCallback := wrapSwitchChat(id)
			button := tb.ReplyButton{
				Text: chatTitle,
			}
			// wrap the callback with the chatID so that the button displays
			// the chat title but calls the switch consumer with the id
			B.Handle(&button, wrappedCallback)
			row = append(row, button)
		}
	}
	keys = append(keys, row)
	pr.Reply = tb.ReplyMarkup{
		ReplyKeyboard:       keys,
		ResizeReplyKeyboard: true,
		OneTimeKeyboard:     true,
	}
}

func wrapSwitchChat(chatID int) func(m *tb.Message) {
	return func(m *tb.Message) {
		m.Text = fmt.Sprintf("%d", chatID)
		ms := []*tb.Message{m}
		switchChat(ms)
	}
}
