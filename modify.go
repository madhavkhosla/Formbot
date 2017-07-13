package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/nlopes/slack"
)

func (f FormBotClient) readAnsAndDisplay() (int, error) {
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
		f.rtm.PostMessage(f.ev.Channel, "", postMessgeParameters)
		return len(answerArray), nil
	}
	return -1, nil
}

func (f FormBotClient) restartFormInSession(userRoutineMap map[string]UserResource, ok bool) (int, error) {
	prefix := fmt.Sprintf("<@%s>", f.infoUserId)
	if f.ev.User != f.infoUserId && strings.HasPrefix(f.ev.Text, fmt.Sprintf("%s create", prefix)) && ok {
		inputStringLength := strings.Split(f.ev.Text, " ")
		Eid = inputStringLength[2]
		if _, err := os.Stat(fmt.Sprintf("/Users/madhav/%s", Eid)); err != nil {
			if os.IsNotExist(err) {
				fmt.Errorf("Error file should exist.\n")
				return -1, err
			}
		}
		return f.readAnsAndDisplay()
	}
	return -1, nil
}
