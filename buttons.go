package main

import (
	tb "gopkg.in/tucnak/telebot.v2"
)

type FunctionButton struct {
	Label    string
	Function func(*tb.Message)
}

func wrapSingleMessage(f func([]*tb.Message) (error)) func(*tb.Message) {
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
	if chatID == 0 {
		BuiltinCommandRegistry["/switchchat"](m)
		return
	}
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
		buttons[0] = buttons[0][:len(buttons[0])-1]
		B.Send(m.Sender, "Check out these commands!", &tb.ReplyMarkup{
			ReplyKeyboard:       buttons,
			ResizeReplyKeyboard: true,
		})
	}
}

var AdminFunctions = []FunctionButton{
	{
		"Add Admin",
		BuiltinCommandRegistry["/addadmin"],
	},
	{
		"Remove Admin",
		BuiltinCommandRegistry["/removeadmin"],
	},
	{
		"View Admins",
		BuiltinCommandRegistry["/viewadmins"],
	},
}

var CommandFunctions = []FunctionButton{
	{
		"Add Command",
		BuiltinCommandRegistry["/addcommand"],
	},
	{
		"Remove Command",
		BuiltinCommandRegistry["/removecommand"],
	},
	{
		"View Commands",
		BuiltinCommandRegistry["/viewcommands"],
	},
}

var ChatFunctions = []FunctionButton{
	{
		"Add Chat",
		BuiltinCommandRegistry["/addchat"],
	},
	{
		"Remove Chat",
		BuiltinCommandRegistry["/removechat"],
	},
	{
		"Switch Chat",
		BuiltinCommandRegistry["/switchchat"],
	},
	{
		"Set Welcome",
		BuiltinCommandRegistry["/setwelcome"],
	},
	{
		"Add Bot to Whitelist",
		BuiltinCommandRegistry["/addwhitelistedbot"],
	},
	{
		"Remove Bot from Whitelist",
		BuiltinCommandRegistry["/removewhitelistedbot"],
	},
}
