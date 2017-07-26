package main

import (
	"bufio"
	"fmt"
	"os"

	"strings"

	"github.com/nlopes/slack"
)

func (f FormBotClient) readAnsAndDisplay(eventChannel string) (int, error) {
	answerArray := make([]slack.AttachmentField, 0, len(questions))
	if file, err := os.Open(fmt.Sprintf("/Users/madhav/%s", Eid)); err == nil {
		defer file.Close()
		var replacer = strings.NewReplacer("$", " ")
		scanner := bufio.NewScanner(file)
		for i := 0; scanner.Scan(); {
			answerArray = append(answerArray, slack.AttachmentField{
				Title: questions[i],
				Value: replacer.Replace(scanner.Text()),
				Short: false,
			})
			i++
		}

		fmt.Println(answerArray)
		if err = scanner.Err(); err != nil {
			f.showError(fmt.Sprintf("ERROR in saving the user input. %v \n", err), eventChannel)
		}
		postMessgeParameters := slack.NewPostMessageParameters()
		postMessgeParameters.AsUser = true
		postMessgeParameters.Attachments = []slack.Attachment{
			{
				Title:  "Intake form filled till now.",
				Color:  "#7CD197",
				Fields: answerArray,
			},
		}
		f.rtm.PostMessage(eventChannel, "", postMessgeParameters)
		return len(answerArray), nil
	}
	return -1, nil
}
