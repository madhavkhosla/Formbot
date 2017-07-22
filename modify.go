package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/nlopes/slack"
)

func (f FormBotClient) readAnsAndDisplay(eventChannel string) (int, error) {
	answerArray := make([]slack.AttachmentField, 0, len(questions))
	if file, err := os.Open(fmt.Sprintf("/Users/madhav/%s", Eid)); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for i := 0; scanner.Scan(); {
			answerArray = append(answerArray, slack.AttachmentField{
				Title: questions[i],
				Value: scanner.Text(),
				Short: false,
			})
			i++
		}
		fmt.Println(answerArray)
		if err = scanner.Err(); err != nil {
			fmt.Errorf("%s", err)
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
