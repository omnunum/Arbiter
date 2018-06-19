package main

import (
	"bytes"
	"encoding/gob"
	"reflect"
	tb "gopkg.in/tucnak/telebot.v2"
)

func EncodePath(p *Path) []byte {
	var by bytes.Buffer
	enc := gob.NewEncoder(&by)
	if err := enc.Encode(&p); err != nil {
		LogE.Printf("could not gob encode %s due to %s",
			reflect.TypeOf(&p), err)
		panic(err)
	}
	data := by.Bytes()
	return data
}

func DecodePath(data []byte) Path {
	var by bytes.Buffer
	by.Write(data)
	dec := gob.NewDecoder(&by)
	p := Path{}
	if err := dec.Decode(&p); err != nil {
		LogE.Printf(
			"Unable to decode data into the new %s struct due to %s",
			reflect.TypeOf(p), err)
	}
	return p
}

func EncodePrompt(pr *Prompt) []byte {
	var by bytes.Buffer
	enc := gob.NewEncoder(&by)
	if err := enc.Encode(&pr); err != nil {
		LogE.Printf("could not gob encode %s due to %s",
			reflect.TypeOf(&pr), err)
		panic(err)
	}
	data := by.Bytes()
	return data
}

func DecodePrompt(data []byte) Prompt {
	var by bytes.Buffer
	by.Write(data)
	dec := gob.NewDecoder(&by)
	pr := Prompt{}
	if err := dec.Decode(&pr); err != nil {
		LogE.Printf(
			"Unable to decode data into the new %s struct due to %s",
			reflect.TypeOf(pr), err)
	}
	return pr
}

func EncodeChat(c *tb.Chat) []byte {
	var by bytes.Buffer
	enc := gob.NewEncoder(&by)
	if err := enc.Encode(&c); err != nil {
		LogE.Printf("could not gob encode %s due to %s",
			reflect.TypeOf(&c), err)
		panic(err)
	}
	data := by.Bytes()
	return data
}

func DecodeChat(data []byte, c *tb.Chat) {
	var by bytes.Buffer
	by.Write(data)
	dec := gob.NewDecoder(&by)
	if err := dec.Decode(&c); err != nil {
		LogE.Printf(
			"Unable to decode data into the new %s struct due to %s",
			reflect.TypeOf(c), err)
		panic(err)
	}
}

func EncodeUser(u *tb.User) []byte {
	var by bytes.Buffer
	enc := gob.NewEncoder(&by)
	if err := enc.Encode(&u); err != nil {
		LogE.Printf("could not gob encode %s due to %s",
			reflect.TypeOf(&u), err)
		panic(err)
	}
	data := by.Bytes()
	return data
}

func DecodeUser(data []byte, u *tb.User) {
	var by bytes.Buffer
	by.Write(data)
	dec := gob.NewDecoder(&by)
	if err := dec.Decode(&u); err != nil {
		LogE.Printf(
			"Unable to decode data into the new %s struct due to %s",
			reflect.TypeOf(u), err)
		panic(err)
	}
}
