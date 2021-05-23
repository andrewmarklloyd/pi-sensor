package main

import (
	"fmt"
	"strings"
)

const (
	delimiter = "|"
)

type Message struct {
	Source    string `json:"source"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type Sensors struct {
	Array []Message `json:"data"`
}

func toString(m Message) string {
	return fmt.Sprintf("%s%s%s%s%s", m.Source, delimiter, m.Status, delimiter, m.Timestamp)
}

func toStruct(s string) Message {
	if s == "" {
		return Message{
			Source:    "",
			Status:    "",
			Timestamp: "",
		}
	} else {
		messageSplit := strings.Split(s, delimiter)
		return Message{
			Source:    messageSplit[0],
			Status:    messageSplit[1],
			Timestamp: messageSplit[2],
		}
	}
}
