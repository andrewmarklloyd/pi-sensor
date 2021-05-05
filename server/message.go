package main

import (
	"fmt"
	"strings"
)

const (
	delimiter = "|"
)

type Message struct {
	Source string `json:"source"`
	Status string `json:"status"`
}

func toString(m Message) string {
	return fmt.Sprintf("%s%s%s", m.Source, delimiter, m.Status)
}

func toStruct(s string) Message {
	messageSplit := strings.Split(s, delimiter)
	return Message{
		Source: messageSplit[0],
		Status: messageSplit[1],
	}
}
