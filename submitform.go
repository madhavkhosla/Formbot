package main

import (
	"encoding/json"
	"strings"

	"fmt"
	"net/url"

	"errors"

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

	json.Unmarshal(evt, &user)

	i := strings.Index(user["body"], "=")
	newStr := user["body"][i+1:]
	str, err := url.QueryUnescape(newStr)
	if err != nil {
		fmt.Errorf("Error")
	}
	fmt.Println(str)
	interactiveRequestMessage := InteractiveMessageRequest{}
	err = json.Unmarshal([]byte(str), &interactiveRequestMessage)
	if err != nil {
		fmt.Errorf(err.Error())
	}
	attachmentAction := interactiveRequestMessage.Actions[0]
	if attachmentAction.Name == "Submit" {
		answers := strings.Split(attachmentAction.Value, "$")
		showOutput := make([]Set, 0, len(questions))
		for i := 0; i < len(questions); i++ {
			showOutput = append(showOutput, Set{Question: questions[i], Answer: answers[i]})
		}
		message, err := json.Marshal(showOutput)
		if err != nil {
			message = []byte("Error in Marshaling json output")
		}
		s := SlackResponse{StatusCode: 200,
			Headers: Header{ContentType: "application/json"},
			Body:    string(message)}

		return s, nil
	} else if attachmentAction.Name == "Select" {

		api := bluele.New("xoxb-207019286820-ZM03MTKKjtiZ6dBdRyMmMJqg")
		questionToModify := attachmentAction.SelectedOptions[0].Value
		channel := interactiveRequestMessage.Channel.ID
		userName := interactiveRequestMessage.User.ID
		chatPostOpts := &bluele.ChatPostMessageOpt{
			AsUser: true,
		}

		err := api.ChatPostMessage(channel, fmt.Sprintf("<@%s> Modify Question %s", userName, questionToModify), chatPostOpts)
		if err != nil {
			panic(err)
		}
		s := SlackResponse{StatusCode: 200,
			Headers: Header{ContentType: "application/json"},
			Body:    ""}

		return s, nil

	}
	return nil, errors.New("Button selected not correct")
}
