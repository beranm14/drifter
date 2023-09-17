package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudtrail"

	log "github.com/sirupsen/logrus"
)

type TerraformState struct {
	Address string                 `json:"address"`
	Values  map[string]interface{} `json:"values"`
}

type PlanOutput struct {
	Type   string                 `json:"type"`
	Change map[string]interface{} `json:"change"`
}

type Event struct {
	EventName string            `json:"EventName"`
	UserId    UserIdentityField `json:"UserIdentity"`
}

type UserIdentityField struct {
	ARN  string `json:"arn"`
	Type string `json:"type"`
}

func loadFileFirst(command string, args []string, file string) string {
	if _, err := os.Stat(file); err == nil {
		data, _ := ioutil.ReadFile(file)
		return string(data)
	} else {
		cmd := exec.Command(command, args...)
		var out, errb bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &errb
		err := cmd.Run()
		if err != nil {
			log.Error("Error executing template: ", err, errb.String())
		}
		ioutil.WriteFile(file, out.Bytes(), 0644)
		return out.String()
	}
}

func getCloudTrailEventsByID(id string, arn string) []Event {
	aws_region := strings.Split(arn, ":")[3]
	if aws_region == "" {
		aws_region = "us-east-1"
	}
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(aws_region),
	}))
	ct := cloudtrail.New(sess)

	input := &cloudtrail.LookupEventsInput{
		LookupAttributes: []*cloudtrail.LookupAttribute{
			{
				AttributeKey:   aws.String("ResourceName"),
				AttributeValue: aws.String(id),
			},
		},
		MaxResults: aws.Int64(1),
	}

	result, err := ct.LookupEvents(input)
	if err != nil {
		log.Error(err)
	}

	var events []Event
	for _, evt := range result.Events {
		var event Event
		json.Unmarshal([]byte(*evt.CloudTrailEvent), &event)
		events = append(events, event)
	}
	return events
}

func getByAddress(state interface{}, address string) map[string]interface{} {
	switch stateType := state.(type) {
	case []interface{}:
		for _, item := range stateType {
			result := getByAddress(item, address)
			if result != nil {
				return result
			}
		}
	case map[string]interface{}:
		if stateType["address"] == address {
			return stateType
		}
		for _, item := range stateType {
			result := getByAddress(item, address)
			if result != nil {
				return result
			}
		}
	}
	return nil
}

func getUsernameFromARN(arn string) string {
	splitted_arn := strings.Split(arn, "/")
	return splitted_arn[len(splitted_arn)-1]
}
