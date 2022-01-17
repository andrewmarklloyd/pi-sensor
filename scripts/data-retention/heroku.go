package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"golang.org/x/oauth2"
)

type HerokuClient struct {
	AppName string
	APIKey  string
}

type HerokuRes struct {
	Id      string `json:"id"`
	Message string `json:"message"`
}

func NewHerokuClient(appName string, apiKey string) HerokuClient {
	return HerokuClient{
		AppName: appName,
		APIKey:  apiKey,
	}
}

func (c *HerokuClient) UpdateToken(token oauth2.Token) error {
	res, err := json.Marshal(token)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(map[string]interface{}{
		"TOKEN_JSON": string(res),
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("https://api.heroku.com/apps/%s/config-vars", c.AppName), bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.heroku+json; version=3")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("performing request to heroku: %s", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading respone body from heroku: %s", err)
	}

	var herokuRes HerokuRes
	err = json.Unmarshal(body, &herokuRes)
	if err != nil {
		return fmt.Errorf("unmarshalling response body from heroku: %s", err)
	}
	if herokuRes.Id == "unauthorized" || herokuRes.Id == "forbidden" {
		return fmt.Errorf("error from heroku: %s", herokuRes.Message)
	}

	return nil
}
