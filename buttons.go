package main

import (
	tb "gopkg.in/tucnak/telebot.v2"
	"encoding/json"
)

type FunctionButton struct {
	Label    string
	Function func(m *tb.Message)
}

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

func wrapPathBegin(p Path) func(m *tb.Message) {
	return func(m *tb.Message) {
		begin(m, p)
	}
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
		"Manage Channels",
		listChannelFunctions,
	},
}


func listCommandFunctions(m *tb.Message) {
	B.Send(m.Sender, "Here are the command management commands", &tb.ReplyMarkup{
		ReplyKeyboard: getReplyKeyboardForCommands(CommandFunctions),
		ResizeReplyKeyboard: true,
	})
}

func listAdminFunctions(m *tb.Message) {
	B.Send(m.Sender, "Here are the admin management commands", &tb.ReplyMarkup{
		ReplyKeyboard: getReplyKeyboardForCommands(AdminFunctions),
		ResizeReplyKeyboard: true,
	})
}

func listChannelFunctions(m *tb.Message) {
	B.Send(m.Sender, "Here are the channel management commands", &tb.ReplyMarkup{
		ReplyKeyboard: getReplyKeyboardForCommands(ChannelFunctions),
		ResizeReplyKeyboard: true,
	})
}

func listFunctionGroups(m *tb.Message) {
	B.Send(m.Sender, "Check out these commands!", &tb.ReplyMarkup{
		ReplyKeyboard: getReplyKeyboardForCommands(FunctionGroups),
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

var ChannelFunctions = []FunctionButton{
	{
		"Add Channel",
		addChannel,
	},
	{
		"Remove Channel",
		wrapPathBegin(Path{
			Prompts: []string{"What channel would you like to beru to stop managing?"},
			Command: "/removeCommand",
		}),
	},
	{
		"View Channels",
		listCommands,
	},
	{
		"Switch Channel",
		listCommands,
	},
}
