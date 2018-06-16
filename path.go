package main

import (
	tb "gopkg.in/tucnak/telebot.v2"
	"encoding/gob"
	"bytes"
	"log"
)

type Path struct {
	Prompts   []string
	Command   string
	Index     int
	Responses []string
}

func Encode(p Path) []byte {
	var by bytes.Buffer
	enc := gob.NewEncoder(&by)
	if err := enc.Encode(p); err != nil {
		log.Printf("could not gob encode path with prompt %s", p.Prompts[0])
		panic(err)
	}
	data := by.Bytes()
	return data
}

func Decode(data []byte) Path {
	var by bytes.Buffer
	by.Write(data)
	dec := gob.NewDecoder(&by)
	p := Path{}
	if err := dec.Decode(&p); err != nil {
		log.Printf("Unable to decode data into the new Path struct due to %s", err)
		panic(err)
	}
	return p
}

func wrapPathBegin(p Path) func(m *tb.Message) {
	return func(m *tb.Message) {
		begin(m, p)
	}
}

var Paths = map[string]Path{
	"addAllAdminsOnChatInvitation": Path{
		Prompts: []string{"Who would you like to add all the admins of this channel?"},
		Command: "/addAdmin",
	},
}
