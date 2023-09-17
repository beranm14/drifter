package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
)

type Drift struct {
	ARN     string
	Address string
}

const slackAPIURL = "https://slack.com/api/chat.postMessage"

type SlackMessage struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

func sendMessageToSlack(channel, messageText string) error {
	slackToken := os.Getenv("SLACK_TOKEN")
	if slackToken == "" {
		return fmt.Errorf("SLACK_TOKEN is not set")
	}

	msg := SlackMessage{
		Channel: channel,
		Text:    messageText,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", slackAPIURL, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+slackToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check if Slack API returned an error
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack API request failed with status code: %d", resp.StatusCode)
	}

	return nil
}

func fetchSlackHandle(awsUsername string) (string, error) {
	// TODO: implement real fetching of slack handle
	slackHandle := os.Getenv("SLACK_HANDLE")
	if slackHandle == "" {
		return "", fmt.Errorf("SLACK_HANDLE is not set")
	}
	return slackHandle, nil
}

func notifyUserAboutDrifts(driftsMap map[string][]Drift) {
	path, _ := os.Getwd()
	for awsUsername, drifts := range driftsMap {
		message := fmt.Sprintf("🔧 following resources in the directory *%s* are drifting and you were the last one making a change, please, take a look:\n", path)
		lvl, ok := os.LookupEnv("LOG_LEVEL")
		if ok && lvl == "debug" {
			awsUsername = "martin_beranek"
		}
		slackUsername, err := fetchSlackHandle(awsUsername)
		if err != nil {
			log.Error("Error executing template: ", err)
			continue
		}
		if slackUsername == "" {
			continue
		}
		for _, item := range drifts {
			message += fmt.Sprintf(" * Address: `%s`, ARN: `%s`\n", item.Address, item.ARN)
		}
		log.Debug("Notifying user %s\n", slackUsername)
		log.Debug(message)
		log.Debug()
		err = sendMessageToSlack(slackUsername, message)
		if err != nil {
			log.Error("Error sending message to Slack: ", err)
		}
	}
}
