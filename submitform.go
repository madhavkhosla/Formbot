package main

import (
	"encoding/json"
	"log"
	"strings"

	"fmt"
	"net/url"

	"errors"

	"strconv"

	"github.com/andygrunwald/go-jira"
	bluele "github.com/bluele/slack"
	"github.com/eawsy/aws-lambda-go/service/lambda/runtime"
	"github.com/nlopes/slack"
)

type InteractiveMessageRequest struct {
	Actions []slack.AttachmentAction
	Channel slack.Channel
	User    slack.User
}

func SubmitForm(evt json.RawMessage, ctx *runtime.Context) (interface{}, error) {
	var user map[string]string

	api := bluele.New("SLACK_TOKEN")

	json.Unmarshal(evt, &user)

	i := strings.Index(user["body"], "=")
	newStr := user["body"][i+1:]
	str, err := url.QueryUnescape(newStr)
	if err != nil {
		log.Printf("Error while decoding the post request %s \n", err.Error())
	}
	log.Printf(str)
	interactiveRequestMessage := InteractiveMessageRequest{}
	err = json.Unmarshal([]byte(str), &interactiveRequestMessage)
	if err != nil {
		log.Printf("Error while un-marshaling request %s \n", err.Error())
	}
	attachmentAction := interactiveRequestMessage.Actions[0]
	if attachmentAction.Name == "Submit" {

		answers := strings.Split(attachmentAction.Value, "$")
		showOutput := make([]Set, 0, len(questions))
		showOutputAttachement := make([]*bluele.Attachment, 0, len(questions))
		for i := 0; i < len(questions); i++ {
			showOutputAttachement = append(showOutputAttachement, &bluele.Attachment{
				Title: questions[i],
				Text:  answers[i],
			})
			showOutput = append(showOutput, Set{Question: questions[i], Answer: answers[i]})
		}

		channel := interactiveRequestMessage.Channel.ID
		userName := interactiveRequestMessage.User.ID
		chatPostOpts := bluele.ChatPostMessageOpt{
			AsUser:      true,
			Attachments: showOutputAttachement,
		}
		err = api.ChatPostMessage(channel, fmt.Sprintf("<@%s> Submitted Form", userName), &chatPostOpts)
		if err != nil {
			log.Printf("Error while posting to slack chat %s\n", err.Error())
		}

		jiraClient, err := jira.NewClient(nil, "https://madhav-test.atlassian.net")
		if err != nil {
			log.Println(err.Error())
		}

		res, err := jiraClient.Authentication.AcquireSessionCookie("madhavnkhosla@gmail.com", "maserati273")
		if err != nil || res == false {
			fmt.Printf("Result: %v\n", res)
			log.Println(err.Error())
		}
		out, err := json.Marshal(showOutput)
		if err != nil {
			log.Println(err.Error())
		}
		i := jira.Issue{
			Fields: &jira.IssueFields{
				Assignee: &jira.User{
					Name: "admin",
				},
				Description: string(out),
				Type: jira.IssueType{
					Name: "Story",
				},
				Project: jira.Project{
					Key: "FOR",
				},
				Summary: "Intake Form",
			},
		}
		issue, _, err := jiraClient.Issue.Create(&i)
		if err != nil {
			log.Println(err.Error())
		}
		fmt.Println(issue)
		s := SlackResponse{StatusCode: 200,
			Headers: Header{ContentType: "application/json"},
			Body:    ""}

		return s, nil
	} else if attachmentAction.Name == "Select" {

		questionToModify := attachmentAction.SelectedOptions[0].Value
		questionNumber, err := strconv.Atoi(questionToModify)
		if err != nil {
			log.Println(err.Error())
		}
		channel := interactiveRequestMessage.Channel.ID
		userName := interactiveRequestMessage.User.ID
		chatPostOpts := &bluele.ChatPostMessageOpt{
			AsUser: true,
		}

		err = api.ChatPostMessage(channel, fmt.Sprintf("<@%s> Modify Question %d", userName, questionNumber+1), chatPostOpts)
		if err != nil {
			log.Printf("Error while posting to slack chat %s\n", err.Error())
		}
		s := SlackResponse{StatusCode: 200,
			Headers: Header{ContentType: "application/json"},
			Body:    ""}

		return s, nil

	}
	return nil, errors.New("Button selected not correct")
}
