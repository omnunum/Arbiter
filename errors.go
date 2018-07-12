package main

import "fmt"

type MissingActiveChatError int

func (e MissingActiveChatError) Error() string {
  tmpl := "no active chat set for user %d"
  return fmt.Sprintf(tmpl, e)
}

type MissingKeyError struct {
  Key string
}

func (e MissingKeyError) Error() string {
  tmpl := "unable to get key %s due to it being unset"
  return fmt.Sprintf(tmpl, e.Key)
}