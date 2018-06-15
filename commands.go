package main

import (
	"log"
	"strings"
	"fmt"
	tb "gopkg.in/tucnak/telebot.v2"
	"time"
)

func begin(m *tb.Message, p Path) {
	key := fmt.Sprintf("%d:activePath", m.Sender.ID)
	err := R.Del(key).Err()
	if err != nil {
		log.Printf("unable to delete active path for %d: %s", m.Sender.ID, err)
		return
	}
	err = R.Set(key, &p, time.Minute).Err()
	if err != nil {
		log.Printf("unable to set active path for %d: %s", m.Sender.ID, err)
		return
	}
	m.Text = ""
	step(m)
}

func step(m *tb.Message) {
	key := fmt.Sprintf("%d:activePath", m.Sender.ID)
	pathData, err := R.Get(key).Result()
	if err != nil {
		log.Printf("attempting to prompt but user %d has no active path: %s", m.Sender.ID, err)
		return
	}
	var p Path
	if err := p.UnmarshalBinary([]byte(pathData)); err != nil {
		log.Printf("Unable to unmarshal data into the new Path struct due to: %s", err)
		return
	}
	if m.Text != "" {
		p.Responses = append(p.Responses, m.Text)
	}
	if p.Index == len(p.Prompts) {
		m.Text = strings.Join(p.Responses, ";")
		log.Printf("reached the end of the path, calling: %s with text: %s", p.Command, m.Text)
		if f, ok := Functions[p.Command]; ok {
			f(m)
		} else {
			log.Printf("function %s cannot be found", p.Command)
		}
		_, err := R.Del(key).Result()
		if err != nil {
			log.Printf("unable to delete active path for user %d: %s", m.Sender.ID, err)
		}
		return
	}
	B.Send(m.Sender, fmt.Sprint(p.Prompts[p.Index]))
	p.Index += 1
	R.Set(key, &p, time.Minute)
}

func registerStaticCommand(userID int, name string, text string) (error) {
	B.Handle(name, func(m *tb.Message) {
		B.Send(m.Sender, text)
	})
	channel, _ := getUsersActiveChannel(userID)
	hashKey := fmt.Sprintf("%s:commands", channel)
	err := R.HSet(hashKey, name, text).Err()
	return err
}

func unregisterStaticCommand(userID int, name string) (error) {
	B.Handle(name, func(m *tb.Message) {})
	channel, _ := getUsersActiveChannel(userID)
	hashKey := fmt.Sprintf("%s:commands", channel)
	err := R.HDel(hashKey, name).Err()
	return err
}

func getUsersActiveChannel(userID int) (string, error) {
	activeChannel, err := R.Get(fmt.Sprintf("activeChannel:%d", userID)).Result()
	if err != nil {
		log.Printf("userID: %d doesn't have an active channel but tried to access it", userID)
		return "", err
	}
	return activeChannel, nil
}

func getKeyboardForCommands(commands map[string]func(m *tb.Message)) ([][]tb.ReplyButton) {
	replyKeys := [][]tb.ReplyButton{}
	replyRow := []tb.ReplyButton{}
	for k, v := range commands {
		replyBtn := tb.ReplyButton{Text: k}
		B.Handle(&replyBtn, v)
		replyRow = append(replyRow, replyBtn)
	}
	replyKeys = append(replyKeys, replyRow)
	return replyKeys
}

func listCommandFunctions(m *tb.Message) {
	B.Send(m.Sender, "Here are the command management commands", &tb.ReplyMarkup{
		ReplyKeyboard: getKeyboardForCommands(CommandFunctions),
	})
}

func listAdminFunctions(m *tb.Message) {
	B.Send(m.Sender, "Here are the admin management commands", &tb.ReplyMarkup{
		ReplyKeyboard: getKeyboardForCommands(AdminFunctions),
	})
}

func listFunctionGroups(m *tb.Message) {
	B.Send(m.Sender, "Check out these commands!", &tb.ReplyMarkup{
		ReplyKeyboard: getKeyboardForCommands(FunctionGroups),
	})
}

func switchChannel(m *tb.Message) {
	split := strings.Split(strings.Replace(m.Text, "/switchChannel ", "", 1), " ")
	channelName := split[0]
	hKey := fmt.Sprintf("activeChannel:%d", m.Sender.ID)
	R.Set(hKey, channelName, 0)
	B.Send(m.Sender, fmt.Sprintf("switched to managing channel: %s", channelName))
}

func addAdmin(m *tb.Message) {
	split := strings.Split(strings.Replace(m.Text, "/addAdmin ", "", 1), " ")
	adminName := split[0]
	channel, _ := getUsersActiveChannel(m.Sender.ID)
	setKey := fmt.Sprintf("%s:admins", channel)
	R.SAdd(setKey, adminName)
	val, _ := R.SMembers(setKey).Result()
	B.Send(m.Sender, fmt.Sprintf("admins now %s", val))
	R.Del(fmt.Sprintf("nextCommand:%d", m.Sender.ID))
}

func removeAdmin(m *tb.Message) {
	split := strings.Split(strings.Replace(m.Text, "/removeAdmin ", "", 1), " ")
	adminName := split[0]
	channel, _ := getUsersActiveChannel(m.Sender.ID)
	setKey := fmt.Sprintf("%s:admins", channel)
	R.SRem(setKey, adminName)
	val, _ := R.SMembers(setKey).Result()
	B.Send(m.Sender, fmt.Sprintf("admins for channel %s now: %s", channel, val))
	R.Del(fmt.Sprintf("nextCommand:%d", m.Sender.ID))
}

func listAdmins(m *tb.Message) {
	channel, _ := getUsersActiveChannel(m.Sender.ID)
	setKey := fmt.Sprintf("%s:admins", channel)
	val, _ := R.SMembers(setKey).Result()
	B.Send(m.Sender, fmt.Sprintf("admins for channel %s: %s", channel, val))
}

func addCommand(m *tb.Message) {
	log.Printf("entered addCommand with msg %s", m.Text)
	split := strings.Split(strings.Replace(m.Text, "/addCommand ", "", 1), ";")
	commandName := split[0]
	if ! strings.HasPrefix(commandName, "/") {
		commandName = "/" + commandName
	}
	commandText := split[1]
	if len(split) == 1 {
		B.Send(m.Sender, fmt.Sprint(
			"you need to specify a command and response to add, such as /addCommand commandName;response text"))
	}
	if err := registerStaticCommand(m.Sender.ID, commandName, commandText); err != nil {
		B.Send(m.Sender, fmt.Sprintf("error while trying to add command: %s", commandName))
	}
	B.Send(m.Sender, fmt.Sprintf("added/overwrote command: %s", commandName))
}

func removeCommand(m *tb.Message) {
	log.Printf("entered removeCommand with msg %s", m.Text)
	commandName := strings.Replace(m.Text, "/removeCommand ", "", 1)
	if ! strings.HasPrefix(commandName, "/") {
		commandName = "/" + commandName
	}
	if err := unregisterStaticCommand(m.Sender.ID, commandName); err != nil {
		B.Send(m.Sender, fmt.Sprintf("error while trying to remove command: %s", commandName))
	}
	B.Send(m.Sender, fmt.Sprintf("removed command: %s", commandName))
}

func listCommands(m *tb.Message) {
	channel, _ := getUsersActiveChannel(m.Sender.ID)
	key := fmt.Sprintf("%s:commands", channel)
	val, _ := R.HKeys(key).Result()
	B.Send(m.Sender, fmt.Sprintf("commands for channel %s: %s", channel, val))
}
