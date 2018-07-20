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
	GAddAdmin GeneratorType = "AddAdminGenerator"
	GRemoveAdmin GeneratorType = "RemoveAdminGenerator"
	GRemoveChat GeneratorType = "RemoveChatGenerator"
)

// a generator takes a message and a prompt, uses the messaage
// to generate output, and writes the output message to the prompt
// Text and or Reply fields.
type Generator func(*tb.Message, *Prompt)

var GeneratorRegistry = map[GeneratorType]Generator{
	GSwitchChat: SwitchChatGenerator,
	GRemoveChat: RemoveChatGenerator,
	GAddAdmin: AddAdminGenerator,
	GRemoveAdmin: RemoveAdminGenerator,
}

func SwitchChatGenerator(m *tb.Message, pr *Prompt) {
	ChatSubGenerator(m, pr, CSwitchChat)
}

func RemoveChatGenerator(m *tb.Message, pr *Prompt) {
	ChatSubGenerator(m, pr, CRemoveChat)
}

func AddAdminGenerator(m *tb.Message, pr *Prompt) {
	AdminSubGenerator(m, pr, CAddAdmin)
}

func RemoveAdminGenerator(m *tb.Message, pr *Prompt) {
	AdminSubGenerator(m, pr, CRemoveAdmin)
}

func AdminSubGenerator(m *tb.Message, pr *Prompt, consumer ConsumerType) {
	userID := m.Sender.ID
	chatID, _, err := getUsersActiveChat(userID)
	if err != nil {
		LogE.Printf("unable to get activeChat: %s", err)
	}
	admins := []string{}
	activeKey := fmt.Sprintf("chat:%d:activeAdmins", chatID)
	if consumer == CAddAdmin{
		if members, err := B.AdminsOf(&tb.Chat{ID: int64(chatID)}); err != nil {
			LogE.Printf("error fetching admins for chat %s", m.Chat.ID)
			B.Send(m.Sender, ErrorResponse)
		} else {
			for _, u := range members {
				encoded := EncodeUser(u.User)
				R.Set(fmt.Sprintf("user:%d:info", u.User.ID),
					encoded,0)
				R.SAdd(fmt.Sprintf("chat:%d:admins", chatID), u.User.ID)
			}
		}
		allKey := fmt.Sprintf("chat:%d:admins", chatID)
		admins, err = R.SDiff(allKey, activeKey).Result()
	} else {
		admins, err = R.SMembers(activeKey).Result()
	}
	// build keyboard for admin selection
	keys := [][]tb.ReplyButton{}
	row := []tb.ReplyButton{}
	viewableAdmins := 0
	for _, a := range admins {
		// convert chatID strings to ints
		if id, err := strconv.Atoi(a); err != nil {
			LogE.Printf("can't convert userID %s into int: %s", a, err)
			pr = &ErrorPrompt
			return
		} else {
			if m.Sender.ID == id {
				continue
			}
			viewableAdmins += 1
			// create a button for each admin
			userName, _ := getUserName(id)
			wrappedCallback := interceptMessageText(fmt.Sprintf("%d", id), consumer)
			button := tb.ReplyButton{
				Text: userName,
			}
			// wrap the callback with the userID so that the button displays
			// the users name but calls the consumer with the id
			B.Handle(&button, wrappedCallback)
			row = append(row, button)
		}
	}
	if viewableAdmins == 0 {
		pr.Text = "You're the only admin."
	} else {
		keys = append(keys, row)
		pr.Reply = tb.ReplyMarkup{
			ReplyKeyboard:       keys,
			ResizeReplyKeyboard: true,
			OneTimeKeyboard:     true,
		}
	}
}

func ChatSubGenerator(m *tb.Message, pr *Prompt, consumer ConsumerType) {
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
			LogE.Printf("can't convert chatID %s into int: %s", c, err)
			pr = &ErrorPrompt
			return
		} else {
			// create a button for each chat
			chatTitle, _ := getChatTitle(id)
			wrappedCallback := interceptMessageText(fmt.Sprintf("%d", id), consumer)
			button := tb.ReplyButton{
				Text: chatTitle,
			}
			// wrap the callback with the chatID so that the button displays
			// the chat title but calls the consumer with the id
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


func interceptMessageText(msg string, consumer ConsumerType) func(m *tb.Message) {
	return func(m *tb.Message) {
		m.Text = msg
		ms := []*tb.Message{m}
		ConsumerRegistry[consumer](ms)
	}
}

var BuiltinCommandRegistry = map[string]func(*tb.Message){
	"/addadmin": wrapPathBegin(Path{
		Prompts: []Prompt{
			{
				GenerateMessage: GAddAdmin,
				Text:            "Who would you like to add as an admin?",
			},
		},
		OwnerOnly: true,
	}),
	"/viewadmins": wrapSingleMessage(viewAdmins),
	"/removeadmin": wrapPathBegin(Path{
		Prompts: []Prompt{
			{
				GenerateMessage: GRemoveAdmin,
				Text:            "Who would you like to remove as an admin?",
			},
		},
		OwnerOnly: true,
	}),
	"/removechat": wrapPathBegin(Path{
		Prompts: []Prompt{
			{
				GenerateMessage: GRemoveChat,
				Text:            "What chat would you like to beru to stop managing?",
			},
		},
	}),
	"/addchat": wrapSingleMessage(addChat),
	"/switchchat": wrapPathBegin(Path{
		Prompts: []Prompt{
			{
				GenerateMessage: GSwitchChat,
				Text:            "What chat would you like to manage?",
			},
		},
	}),
	"/addcommand": wrapPathBegin(Path{
		Prompts: []Prompt{
			{Text: "What's the name of the command?",},
			{Text: "What would you like the response to be? (Markdown formatting is supported)",},
		},
		Consumer: CAddCommand,
	}),
	"/removecommand": wrapPathBegin(Path{
		Prompts: []Prompt{
			{Text: "What command would you like to remove?"},
		},
		Consumer: CRemoveCommand,
	}),
	"/viewcommands": wrapSingleMessage(ConsumerRegistry[CViewCommands]),
	"/setwelcome": wrapPathBegin(Path{
		Prompts: []Prompt{
			{Text: `What is the message you would like to welcome your users with?
(you can use $username to be replaced with the new members username)`,},
			{Text: "How many users do you want to join between each welcome message?",},
		},
		Consumer: CSetWelcome,
	}),
	"/togglejoinmsg": wrapSingleMessage(ConsumerRegistry[CToggleJoinMessage]),
}
