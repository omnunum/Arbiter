package main

import (
	tb "gopkg.in/tucnak/telebot.v2"
	"fmt"
	"strconv"
	)


func SwitchChatPrompt(m *tb.Message, pr *Prompt) {
	userID := m.Sender.ID
	k := fmt.Sprintf("user:%d:chats", userID)
	chatIDs, err := R.SMembers(k).Result()
	if err != nil {
		LogE.Printf("couldn't get chat IDs for %s: %s", userID, err)
		pr = &ErrorPrompt
		return
	}
	keys := [][]tb.ReplyButton{}
	row := []tb.ReplyButton{}
	for _, c := range chatIDs {
		if id, err := strconv.Atoi(c); err != nil {
			LogE.Printf("can't convert userID %s into int: %s", c, err)
			pr = &ErrorPrompt
			return
		} else {
			chatTitle, _ := getChatTitle(id)
			wrappedCallback := wrapSwitchChat(id)
			button := tb.ReplyButton{
				Text: chatTitle,
			}
			B.Handle(&button, wrappedCallback)
			row = append(row, button)
		}
	}
	keys = append(keys, row)
	pr.Reply = tb.ReplyMarkup{
		ReplyKeyboard:       keys,
		ResizeReplyKeyboard: true,
		OneTimeKeyboard: true,
	}
}

func wrapSwitchChat(chatID int) func(m *tb.Message) {
	return func(m *tb.Message) {
		m.Text = fmt.Sprintf("%d", chatID)
		ms := []*tb.Message{m}
		switchChat(ms)
	}
}