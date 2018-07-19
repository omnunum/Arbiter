package main

import (
	tb "gopkg.in/tucnak/telebot.v2"
)

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
	"/viewcommands": wrapSingleMessage(viewCommands),
	"/setwelcome": wrapPathBegin(Path{
		Prompts: []Prompt{
			{Text: `What is the message you would like to welcome your users with?
(you can use $username to be replaced with the new members username)`,},
			{Text: "How many users do you want to join between each welcome message?",},
		},
		Consumer: CSetWelcome,
	}),
}
