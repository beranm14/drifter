package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

func init() {
	lvl, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		lvl = "info"
	}
	ll, err := log.ParseLevel(lvl)
	if err != nil {
		ll = log.DebugLevel
	}
	log.SetLevel(ll)
}

func main() {
	terraformInterpreter := os.Getenv("TERRAFORM_INTERPRETER")
	if terraformInterpreter == "" {
		terraformInterpreter = "terragrunt"
	}

	driftsMap := make(map[string][]Drift)

	stateData := loadFileFirst(terraformInterpreter, []string{"show", "-json"}, "terraform-show-state.json")
	var state interface{}
	json.Unmarshal([]byte(stateData), &state)

	planOutput := loadFileFirst(terraformInterpreter, []string{"plan", "-json"}, "terraform.tfplan.json")
	for _, line := range strings.Split(planOutput, "\n") {
		var lineData PlanOutput
		err := json.Unmarshal([]byte(line), &lineData)
		if err != nil {
			continue
		}
		if lineData.Type == "planned_change" {
			address := lineData.Change["resource"].(map[string]interface{})["addr"].(string)
			resource := getByAddress(state, address)
			if resource == nil {
				continue
			}
			if resource["values"].(map[string]interface{})["arn"] == nil || resource["values"].(map[string]interface{})["id"] == nil {
				continue
			}
			id := resource["values"].(map[string]interface{})["id"].(string)
			arn := resource["values"].(map[string]interface{})["arn"].(string)
			events := getCloudTrailEventsByID(id, arn)
			if len(events) > 0 {
				event := events[0]
				username := getUsernameFromARN(event.UserId.ARN)
				fmt.Printf("Drift resource %s with address %s, last one touching it was %s, did %s\n", id, address, username, event.EventName)
				driftsMap[username] = append(driftsMap[username], Drift{
					ARN:     arn,
					Address: address,
				})
			}
		}
	}
	notifyUserAboutDrifts(driftsMap)
}
