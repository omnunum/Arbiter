package main

import (
	tb "gopkg.in/tucnak/telebot.v2"
)

type FunctionButton struct {
	Label    string
	Function func(*tb.Message)
}

func wrapSingleMessage(f func([]*tb.Message)) func(*tb.Message) {
	return func(m *tb.Message) {
		f([]*tb.Message{m})
	}
}

var FunctionGroups = []FunctionButton{
	{
		"Manage Commands",
		listCommandFunctions,
	},
	{
		"Manage Chats",
		listChatFunctions,
	},
	{
		"Manage Admins",
		listAdminFunctions,
	},
}

func listCommandFunctions(m *tb.Message) {
	B.Send(m.Sender, "Here are the command management commands", &tb.ReplyMarkup{
		ReplyKeyboard:       getReplyKeyboardForCommands(CommandFunctions),
		ResizeReplyKeyboard: true,
	})
}

func listAdminFunctions(m *tb.Message) {
	B.Send(m.Sender, "Here are the admin management commands", &tb.ReplyMarkup{
		ReplyKeyboard:       getReplyKeyboardForCommands(AdminFunctions),
		ResizeReplyKeyboard: true,
	})
}

func listChatFunctions(m *tb.Message) {
	B.Send(m.Sender, "Here are the chat management commands", &tb.ReplyMarkup{
		ReplyKeyboard:       getReplyKeyboardForCommands(ChatFunctions),
		ResizeReplyKeyboard: true,
	})
}

func listFunctionGroups(m *tb.Message) {
	chatID, _, _ := getUsersActiveChat(m.Sender.ID)

	buttons := getReplyKeyboardForCommands(FunctionGroups)
	// if the user is a chat owner
	if access, err := userHasAdminManagementAccess(m.Sender.ID, chatID); access {
		B.Send(m.Sender, "Check out these commands!", &tb.ReplyMarkup{
			ReplyKeyboard:       buttons,
			ResizeReplyKeyboard: true,
		})
		// if the user isn't an owner
	} else if !access && err == nil {
		// remove last element (admin buttons)
		buttons = buttons[:len(buttons)-1]
		B.Send(m.Sender, "Check out these commands!", &tb.ReplyMarkup{
			ReplyKeyboard:       buttons,
			ResizeReplyKeyboard: true,
		})
	}
}

var AdminFunctions = []FunctionButton{
	{
		"Add Admin",
		wrapPathBegin(Path{
			Prompts: []Prompt{
				{Text: "Who would you like to add as an admin?"},
			},
			Consumer: CAddAdmin,
		}),
	},
	{
		"Remove Admin",
		wrapPathBegin(Path{
			Prompts: []Prompt{
				{Text: "Who would you like to remove as an admin?"},
			},
			Consumer: CRemoveAdmin,
		}),
	},
	{
		"View Admins",
		wrapSingleMessage(viewAdmins),
	},
}

var CommandFunctions = []FunctionButton{
	{
		"Add Command",
		wrapPathBegin(Path{
			Prompts: []Prompt{
				{Text: "What's the name of the command?",},
				{Text: "What would you like the response to be? (Markdown formatting is supported)",},
			},
			Consumer: CAddAdmin,
		}),
	},
	{
		"Remove Command",
		wrapPathBegin(Path{
			Prompts: []Prompt{
				{Text: "What command would you like to remove?"},
			},
			Consumer: CRemoveCommand,
		}),
	},
	{
		"View Commands",
		wrapSingleMessage(viewCommands)},
}

var ChatFunctions = []FunctionButton{
	{
		"Add Chat",
		wrapSingleMessage(addChat),
	},
	{
		"Remove Chat",
		wrapPathBegin(Path{
			Prompts: []Prompt{
				{Text: "What chat would you like to beru to stop managing?"},
			},
			Consumer: CRemoveCommand,
		}),
	},
	{
		"Switch Chat",
		wrapPathBegin(Path{
			Prompts: []Prompt{
				{
					GenerateMessage: GSwitchChat,
					Text:            "What chat would you like to manage?",
				},
			},
		}),
	},
}
