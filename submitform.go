package main

import (
	"encoding/json"
	"strings"

	"github.com/eawsy/aws-lambda-go/service/lambda/runtime"
)

func SubmitForm(evt json.RawMessage, ctx *runtime.Context) (interface{}, error) {
	var user map[string]string

	json.Unmarshal(evt, &user)
	text := GetStringInBetween(user["body"], "FS", "FE")
	var replacer = strings.NewReplacer("+", " ")
	text = replacer.Replace(text)
	answers := strings.Split(text, "%24")
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
}

func GetStringInBetween(str string, start string, end string) (result string) {
	s := strings.Index(str, start)
	if s == -1 {
		return
	}
	s += len(start)
	e := strings.Index(str, end)
	return str[s:e]
}
