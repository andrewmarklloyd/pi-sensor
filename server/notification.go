package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Messenger struct {
	AccountSID string
	AuthToken  string
	To         string
	From       string
}

func newMessenger(twilioConfig TwilioConfig) Messenger {
	return Messenger{
		twilioConfig.accountSID,
		twilioConfig.authToken,
		twilioConfig.to,
		twilioConfig.from,
	}
}

func (m Messenger) SendMessage(body string) (string, error) {
	urlStr := "https://api.twilio.com/2010-04-01/Accounts/" + m.AccountSID + "/Messages.json"
	msgData := url.Values{}
	msgData.Set("To", m.To)
	msgData.Set("From", m.From)
	msgData.Set("Body", body)
	msgDataReader := *strings.NewReader(msgData.Encode())

	client := &http.Client{}
	req, _ := http.NewRequest("POST", urlStr, &msgDataReader)
	req.SetBasicAuth(m.AccountSID, m.AuthToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, _ := client.Do(req)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var data map[string]interface{}
		decoder := json.NewDecoder(resp.Body)
		err := decoder.Decode(&data)
		if err != nil {
			return "", err
		}
		return data["sid"].(string), nil
	} else {
		return "", fmt.Errorf("Response code: %s", resp.Status)
	}
}
