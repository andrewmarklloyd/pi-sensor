package notification

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/config"
)

type Messenger struct {
	TwilioConfig config.TwilioConfig
}

func NewMessenger(twilioConfig config.TwilioConfig) Messenger {
	return Messenger{
		TwilioConfig: twilioConfig,
	}
}

func (m Messenger) SendMessage(body string) (string, error) {
	urlStr := "https://api.twilio.com/2010-04-01/Accounts/" + m.TwilioConfig.AccountSID + "/Messages.json"
	msgData := url.Values{}
	msgData.Set("To", m.TwilioConfig.To)
	msgData.Set("From", m.TwilioConfig.From)
	msgData.Set("Body", body)
	msgDataReader := *strings.NewReader(msgData.Encode())

	client := &http.Client{}
	req, _ := http.NewRequest("POST", urlStr, &msgDataReader)
	req.SetBasicAuth(m.TwilioConfig.AccountSID, m.TwilioConfig.AuthToken)
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

func (m Messenger) CheckBalance() error {
	limit := 0.80
	urlStr := "https://api.twilio.com/2010-04-01/Accounts/" + m.TwilioConfig.AccountSID + "/Balance.json"
	msgData := url.Values{}
	msgDataReader := *strings.NewReader(msgData.Encode())

	client := &http.Client{}
	req, _ := http.NewRequest("GET", urlStr, &msgDataReader)
	req.SetBasicAuth(m.TwilioConfig.AccountSID, m.TwilioConfig.AuthToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	var data map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&data)
	if err != nil {
		return err
	}

	balance, err := strconv.ParseFloat(data["balance"].(string), 8)
	if err != nil {
		return err
	}

	if balance < limit {
		return fmt.Errorf(fmt.Sprintf("WARNING: Twilio balance %f is less than limit %f", balance, limit))
	}

	return nil
}