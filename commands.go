package main

import (
	"log"
	"strings"
	"fmt"
	"github.com/go-redis/redis"
	tb "gopkg.in/tucnak/telebot.v2"
		"time"
)


func begin(client *redis.Client, b *tb.Bot, m *tb.Message, p Path) {
	key := fmt.Sprintf("%d:activePath", m.Sender.ID)
	err := client.Del(key).Err()
	if err != nil {
		log.Printf("unable to delete active path for %d: %s", m.Sender.ID, err)
		return
	}
	err = client.Set(key, &p, time.Minute).Err()
	if err != nil {
		log.Printf("unable to set active path for %d: %s", m.Sender.ID, err)
		return
	}
	m.Text = ""
	step(client, b, m)
}

func step(client *redis.Client, b *tb.Bot, m *tb.Message) {
	key := fmt.Sprintf("%d:activePath", m.Sender.ID)
	pathData, err := client.Get(key).Result()
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
		Functions[p.Command](m)
		_, err := client.Del(key).Result()
		if err != nil {
			log.Printf("unable to delete active path for user %d: %s", m.Sender.ID, err)
		}
		return
	}
	b.Send(m.Sender, fmt.Sprint(p.Prompts[p.Index]))
	p.Index += 1
	client.Set(key, &p, time.Minute)
}

func getUsersActiveChannel(client *redis.Client, userID int) (string, error) {
	activeChannel, err := client.Get(fmt.Sprintf("activeChannel:%d", userID)).Result()
	if err != nil {
		log.Printf("userID: %d doesn't have an active channel but tried to access it", userID)
		return "", err
	}
	return activeChannel, nil
}

func listCommandCommands(client *redis.Client, b *tb.Bot, m *tb.Message) {
	// This button will be displayed in user's
	// reply keyboard.
	replyKeys := [][]tb.ReplyButton{}
	replyRow := []tb.ReplyButton{}
	for k, v := range CommandCommands {
		replyBtn := tb.ReplyButton{Text: k}
		b.Handle(&replyBtn, v)
		replyRow = append(replyRow, replyBtn)
	}
	replyKeys = append(replyKeys, replyRow)
	b.Send(m.Sender, "Here are the command management commands", &tb.ReplyMarkup{
		ReplyKeyboard:  replyKeys,
	})
}

func listAdminCommands(client *redis.Client, b *tb.Bot, m *tb.Message) {
	// This button will be displayed in user's
	// reply keyboard.
	replyKeys := [][]tb.ReplyButton{}
	replyRow := []tb.ReplyButton{}
	for k, v := range AdminCommands {
		replyBtn := tb.ReplyButton{Text: k}
		b.Handle(&replyBtn, v)
		replyRow = append(replyRow, replyBtn)
	}
	replyKeys = append(replyKeys, replyRow)
	b.Send(m.Sender, "Here are the admin management commands", &tb.ReplyMarkup{
		ReplyKeyboard:  replyKeys,
	})
}

func listCommandGroups(client *redis.Client, b *tb.Bot, m *tb.Message) {
	// This button will be displayed in user's
	// reply keyboard.
	replyKeys := [][]tb.ReplyButton{}
	for k, v := range FunctionGroups {
		replyBtn := tb.ReplyButton{Text: k}
		b.Handle(&replyBtn, v)
		replyKeys = append(replyKeys, []tb.ReplyButton{replyBtn})
	}
	b.Send(m.Sender, "Check out these commands!", &tb.ReplyMarkup{
		ReplyKeyboard:  replyKeys,
	})
}

func switchChannel(client *redis.Client, b *tb.Bot, m *tb.Message) {
	split := strings.Split(strings.Replace(m.Text, "/switchChannel ", "", 1), " ")
	channelName := split[0]
	hKey := fmt.Sprintf("activeChannel:%d", m.Sender.ID)
	client.Set(hKey, channelName, 0)
	b.Send(m.Sender, fmt.Sprintf("switched to managing channel: %s", channelName))
}

func addAdmin(client *redis.Client, b *tb.Bot, m *tb.Message) {
	split := strings.Split(strings.Replace(m.Text, "/addAdmin ", "", 1), " ")
	adminName := split[0]
	channel, _ := getUsersActiveChannel(client, m.Sender.ID)
	setKey := fmt.Sprintf("%s:admins", channel)
	client.SAdd(setKey, adminName)
	val, _ := client.SMembers(setKey).Result()
	b.Send(m.Sender, fmt.Sprintf("admins now %s", val))
	client.Del(fmt.Sprintf("nextCommand:%d", m.Sender.ID))
}

func removeAdmin(client *redis.Client, b *tb.Bot, m *tb.Message) {
	split := strings.Split(strings.Replace(m.Text, "/removeAdmin ", "", 1), " ")
	adminName := split[0]
	channel, _ := getUsersActiveChannel(client, m.Sender.ID)
	setKey := fmt.Sprintf("%s:admins", channel)
	client.SRem(setKey, adminName)
	val, _ := client.SMembers(setKey).Result()
	b.Send(m.Sender, fmt.Sprintf("admins for channel %s now: %s", channel, val))
	client.Del(fmt.Sprintf("nextCommand:%d", m.Sender.ID))
}

func viewAdmins(client *redis.Client, b *tb.Bot, m *tb.Message) {
	channel, _ := getUsersActiveChannel(client, m.Sender.ID)
	setKey := fmt.Sprintf("%s:admins", channel)
	val, _ := client.SMembers(setKey).Result()
	b.Send(m.Sender, fmt.Sprintf("admins for channel %s: %s", channel, val))
}

func addCommand(client *redis.Client, b *tb.Bot, m *tb.Message) {
	split := strings.Split(strings.Replace(m.Text, "/addCommand ", "", 1), ";")
	if len(split) == 1 {
		b.Send(m.Sender, fmt.Sprint(
			"you need to specify a command and response to add, such as /addCommand commandName;response text"))
	}
	commandName := split[0]
	commandText := split[1]
	channel, _ := getUsersActiveChannel(client, m.Sender.ID)
	hashKey := fmt.Sprintf("%s:commands", channel)
	client.HSet(hashKey, commandName, commandText)
	val, _ := client.HKeys(hashKey).Result()
	b.Send(m.Sender, fmt.Sprintf("commands for channel %s now: %s", channel, val))
}

func removeCommand(client *redis.Client, b *tb.Bot, m *tb.Message) {
	commandName := strings.Split(m.Text, " ")[1]
	// commandText := strings.Split(m.Text, " ")[2]
	channel, _ := getUsersActiveChannel(client, m.Sender.ID)
	hashKey := fmt.Sprintf("%s:commands", channel)
	client.SRem(hashKey, commandName)
	val, _ := client.SMembers(hashKey).Result()
	b.Send(m.Sender, fmt.Sprintf("commands for channel %s now: %s", channel, val))
}

