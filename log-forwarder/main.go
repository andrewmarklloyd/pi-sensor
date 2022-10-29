package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type logMessage struct {
	Message string `json:"message"`
}

type syslog struct {
	Identifier string `json:"SYSLOG_IDENTIFIER"`
	Message    string `json:"MESSAGE"`
	Error      error
}

type response struct {
	Error string `json:"error"`
}

func main() {
	logEndpoint := os.Getenv("LOG_ENDPOINT")
	if logEndpoint == "" {
		fmt.Println("LOG_ENDPOINT env var must be set")
		os.Exit(1)
	}
	logApiKey := os.Getenv("LOG_API_KEY")
	if logApiKey == "" {
		fmt.Println("LOG_API_KEY env var must be set")
		os.Exit(1)
	}

	logChannel := make(chan syslog)
	go tailSystemdLogs(logChannel)
	for log := range logChannel {
		if log.Error != nil {
			fmt.Println(fmt.Sprintf("error receiving logs from journalctl channel: %s", log.Error))
			break
		}

		err := sendLogs(log.Message, logEndpoint, logApiKey)
		if err != nil {
			fmt.Println("error sending logs:", err)
		}
	}
}

func tailSystemdLogs(ch chan syslog) error {
	cmd := exec.Command("journalctl", "-u", "pi-sensor-agent-v2", "-f", "-n 0", "--output", "json")
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating command stdout pipe: %s", err)
	}

	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			var s syslog
			if err := json.Unmarshal([]byte(scanner.Text()), &s); err != nil {
				s.Error = fmt.Errorf("unmarshalling log: %s, original log text: %s", err, scanner.Text())
				ch <- s
				break
			}
			if s.Message != "" && s.Identifier != "systemd" && !strings.Contains(s.Message, "Logs begin at") {
				ch <- s
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting command: %s", err)
	}

	if err := cmd.Wait(); err != nil {
		close(ch)
		return fmt.Errorf("waiting for command: %s", err)
	}

	return nil
}

func sendLogs(log, logEndpoint, logApiKey string) error {
	req, err := http.NewRequest("POST", logEndpoint, bytes.NewBuffer([]byte(log)))

	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Add("api-key", logApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var res response
	err = json.Unmarshal(body, &res)
	if err != nil {
		return err
	}

	if res.Error != "" {
		return fmt.Errorf(res.Error)
	}

	return nil
}
