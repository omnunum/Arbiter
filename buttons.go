package main

import (
	tb "gopkg.in/tucnak/telebot.v2"
)

type FunctionButton struct {
	Label    string
	Function func(m *tb.Message)
}

var FunctionGroups = []FunctionButton{
	{
		"Manage Commands",
		listCommandFunctions,
	},
	{
		"Manage Admins",
		listAdminFunctions,
	},
	{
		"Manage Chats",
		listChatFunctions,
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
	B.Send(m.Sender, "Check out these commands!", &tb.ReplyMarkup{
		ReplyKeyboard:       getReplyKeyboardForCommands(FunctionGroups),
		ResizeReplyKeyboard: true,
	})
}

var AdminFunctions = []FunctionButton{
	{
		"Add Admin",
		wrapPathBegin(Path{
			Prompts: []string{"Who would you like to add as an admin?"},
			Command: "/addAdmin",
		}),
	},
	{
		"Remove Admin",
		wrapPathBegin(Path{
			Prompts: []string{"Who would you like to remove as an admin?"},
			Command: "/removeAdmin",
		}),
	},
	{
		"View Admins",
		listAdmins,
	},
}

var CommandFunctions = []FunctionButton{
	{
		"Add Command",
		wrapPathBegin(Path{
			Prompts: []string{"What's the name of the command?",
				"What would you like the response to be? (Markdown formatting is supported)"},
			Command: "/addCommand",
		}),
	},
	{
		"Remove Command",
		wrapPathBegin(Path{
			Prompts: []string{"What command would you like to remove?"},
			Command: "/removeCommand",
		}),
	},
	{
		"View Commands",
		listCommands},
}

var ChatFunctions = []FunctionButton{
	{
		"Add Chat",
		addChat,
	},
	{
		"Remove Chat",
		wrapPathBegin(Path{
			Prompts: []string{"What chat would you like to beru to stop managing?"},
			Command: "/removeCommand",
		}),
	},
	{
		"View Chats",
		listCommands,
	},
	{
		"Switch Chat",
		listCommands,
	},
}
