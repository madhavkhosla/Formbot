package main

import (
	"fmt"
	"strings"

	"bufio"
	"os"

	"github.com/nlopes/slack"
)

func (f FormBotClient) startForm(userRoutineMap map[string]*UserResource, ok bool) {
	prefix := fmt.Sprintf("<@%s>", f.infoUserId)
	if f.ev.User != f.infoUserId && strings.HasPrefix(f.ev.Text, fmt.Sprintf("%s create", prefix)) && !ok {
		fmt.Println("Inside Start form")
		inputStringLength := strings.Split(f.ev.Text, " ")
		Eid = inputStringLength[2]
		trigger := make(chan int)
		if _, err := os.Stat(fmt.Sprintf("/Users/madhav/%s", Eid)); err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Inside File does not exists")
				// file does not exist
				file, err := os.Create(fmt.Sprintf("/Users/madhav/%s", Eid))
				if err != nil {
					fmt.Errorf("ERROR in creating a file \n")
				}
				w := bufio.NewWriter(file)
				userRoutineMap[f.ev.User] = &UserResource{trigger, w, []int{}}
				fmt.Println(userRoutineMap)
				go f.sendQuestions(trigger, userRoutineMap, 0)
			}
		} else {
			fmt.Println("File already exists")
			lastAnsSaved, err := f.readAnsAndDisplay()
			if err != nil {
				fmt.Errorf("Something went wrong when file already exists and starting form")
			}
			file, err := os.OpenFile(fmt.Sprintf("/Users/madhav/%s", Eid), os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				fmt.Errorf("ERROR in opening a file \n")
			}
			w := bufio.NewWriter(file)
			userRoutineMap[f.ev.User] = &UserResource{trigger, w, []int{}}
			fmt.Println(userRoutineMap)
			go f.sendQuestions(trigger, userRoutineMap, lastAnsSaved)
		}
	}
}

func (f FormBotClient) invalidCreateCommand() bool {
	prefix := fmt.Sprintf("<@%s>", f.infoUserId)
	if f.ev.User != f.infoUserId && strings.HasPrefix(f.ev.Text, fmt.Sprintf("%s create", prefix)) {
		inputStringLength := strings.Split(f.ev.Text, " ")

		if len(inputStringLength) != 3 {
			f.rtm.SendMessage(f.rtm.NewOutgoingMessage(fmt.Sprintf("Invalid input command"), f.ev.Channel))
			return false
		}
	}
	return true
}

func (f FormBotClient) helpCommands() {
	prefix := fmt.Sprintf("<@%s>", f.infoUserId)
	if f.ev.User != f.infoUserId && (f.ev.Text == prefix || f.ev.Text == fmt.Sprintf("%s help", prefix)) {
		postMessgeParameters := slack.NewPostMessageParameters()
		postMessgeParameters.Attachments = []slack.Attachment{
			{
				Title: "Command to start new intake form",
				Text:  "@formbot create [EID]",
				Color: "#7CD197",
			},
		}
		f.rtm.PostMessage(f.ev.Channel, fmt.Sprintf("Formbot help commands"), postMessgeParameters)
	}
}
