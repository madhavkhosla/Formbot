package main

import (
	"fmt"
	"strings"

	"os"

	"strconv"

	"github.com/nlopes/slack"
)

func (formBotClient FormBotClient) startUserRoutine(existingUserResource *UserResource, channel string) {
	for {
		select {
		case userEvent := <-existingUserResource.UserChannel:
			userInputArray := []byte(userEvent.Text)
			userInputArray = append(userInputArray, '$')
			outputArray := make([]byte, 100)
			for i := 0; i < 99; i++ {
				if i > len(userInputArray)-1 {
					outputArray[i] = 0
					continue
				}
				outputArray[i] = userInputArray[i]
			}
			outputArray[99] = '\n'
			n3, err := existingUserResource.File.Write(outputArray)
			if err != nil {
				formBotClient.showError(fmt.Sprintf("ERROR in saving the user input. %v \n", err), channel)
			}
			fmt.Println(existingUserResource)
			fmt.Printf("wrote %s  %d bytes\n", userEvent.Text, n3)
			existingUserResource.SyncChannel <- -1
		case <-existingUserResource.QuitChannel:
			fmt.Println("quit")
			return
		}
	}
}

func (formBotClient FormBotClient) startForm(ev *slack.MessageEvent, userFullMap map[string]map[string]*UserResource,
	userRoutineMap map[string]*UserResource) {

	fmt.Println("Inside Start form")
	trigger := make(chan int)
	userChannel := make(chan *slack.MessageEvent)
	modifyChannel := make(chan *slack.MessageEvent)
	inputStringLength := strings.Split(ev.Text, " ")
	Eid := inputStringLength[2]
	existingUserResource, ok := userFullMap[ev.User][Eid]

	if !ok {
		innerMap := make(map[string]*UserResource)
		// 1) Check if file exists or a new file
		if _, err := os.Stat(fmt.Sprintf("/tmp/%s", Eid)); err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Inside File does not exists")
				// file does not exist
				file, err := os.Create(fmt.Sprintf("/tmp/%s", Eid))
				if err != nil {
					formBotClient.showError(fmt.Sprintf("ERROR in creating a file. %v \n", err), ev.Channel)
				}
				innerMap[Eid] = &UserResource{userChannel,
					modifyChannel,
					trigger,
					file,
					make(chan int),
					Eid,
					false}
				userFullMap[ev.User] = innerMap
				userRoutineMap[ev.User] = userFullMap[ev.User][Eid]
				fmt.Println(userFullMap)
				go formBotClient.sendQuestions(ev, trigger, userFullMap, 0, Eid)
			}
		} else {
			fmt.Println("File already exists")
			lastAnsSaved, err := formBotClient.readAnsAndDisplay(ev.Channel, Eid)
			if err != nil {
				formBotClient.showError(fmt.Sprintf("Form already exists. Something went wrong in displaying it. %v \n", err), ev.Channel)
			}
			file, err := os.OpenFile(fmt.Sprintf("/tmp/%s", Eid), os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				formBotClient.showError(fmt.Sprintf("Form already exists. Something went wrong in opening it. %v \n", err), ev.Channel)
			}
			innerMap[Eid] = &UserResource{userChannel,
				modifyChannel,
				trigger,
				file,
				make(chan int),
				Eid,
				false}
			userFullMap[ev.User] = innerMap
			userRoutineMap[ev.User] = userFullMap[ev.User][Eid]
			fmt.Println(userFullMap)
			go formBotClient.sendQuestions(ev, trigger, userFullMap, lastAnsSaved, Eid)
		}
		go formBotClient.startUserRoutine(userRoutineMap[ev.User], ev.Channel)
	} else {
		fmt.Println("Existing user restoring older form")
		lastQuestionAsked, err := formBotClient.readAnsAndDisplay(ev.Channel, Eid)
		if err != nil {
			formBotClient.showError(fmt.Sprintf("Form already exists. Something went wrong in displaying it. %v \n", err), ev.Channel)
		}
		// If someone calls modify and then switches forms, when he comes back to first form
		// Modify flag is removed.
		existingUserResource.Modify = false
		userRoutineMap[ev.User] = existingUserResource
		if lastQuestionAsked >= len(questions) {
			formBotClient.submitForm(ev, existingUserResource)
			return
		}
		if lastQuestionAsked >= 0 {
			userRoutineMap[ev.User].SyncChannel <- lastQuestionAsked
		}
	}
}

func (f FormBotClient) invalidCreateCommand(ev *slack.MessageEvent) bool {
	inputStringLength := strings.Split(ev.Text, " ")
	if len(inputStringLength) != 3 {

		postMessgeParameters := slack.NewPostMessageParameters()
		postMessgeParameters.Attachments = []slack.Attachment{
			{
				Title: "Incorrect input command for creating intake form",
				Text:  "Please try the help command: @formbot help",
				Color: "#7CD197",
			},
		}
		f.rtm.PostMessage(ev.Channel, "", postMessgeParameters)

		return false
	}
	return true
}

func (f FormBotClient) helpCommands(ev *slack.MessageEvent) bool {
	botId := f.rtm.GetInfo().User.ID
	prefix := fmt.Sprintf("<@%s>", botId)
	if ev.User != botId && (ev.Text == prefix || ev.Text == fmt.Sprintf("%s help", prefix)) {
		postMessgeParameters := slack.NewPostMessageParameters()
		postMessgeParameters.Attachments = []slack.Attachment{
			{
				Title: "Command to start or restore an intake form",
				Text:  "@formbot create [EID]",
				Color: "#7CD197",
			},
			{
				Title: "Command to modify a question once form is started",
				Text:  "@formbot modify",
				Color: "#7CD197",
			},
		}
		f.rtm.PostMessage(ev.Channel, fmt.Sprintf("Formbot help commands"), postMessgeParameters)
		return true
	}
	return false
}

func (f FormBotClient) updateAnswer(ev *slack.MessageEvent, existingUserResource *UserResource) {
	inputStringLength := strings.Split(ev.Text, " ")
	modifyQuestionString := inputStringLength[3]
	modifyQuestion, err := strconv.Atoi(modifyQuestionString)
	modifyQuestion = modifyQuestion - 1
	if err != nil {
		f.showError(fmt.Sprintf("Error in update Answer in converting question number to int. %v \n", err), ev.Channel)
	}
	postMessgeParameters := slack.NewPostMessageParameters()
	postMessgeParameters.Attachments = []slack.Attachment{
		{
			Title: questions[modifyQuestion],
			Color: "#7CD197",
		},
	}
	f.rtm.PostMessage(ev.Channel, "Please provide answer for question", postMessgeParameters)

	go f.modifyAnswerRoutine(modifyQuestion, existingUserResource, ev.Channel)
}

func (f FormBotClient) modifyAnswerRoutine(modifyQuestion int, existingUserResource *UserResource, channel string) {

	modifyAnsEvent := <-existingUserResource.ModifyChannel

	userInputArray := []byte(modifyAnsEvent.Text)
	userInputArray = append(userInputArray, '$')
	outputArray := make([]byte, 100)
	for i := 0; i < 99; i++ {
		if i > len(userInputArray)-1 {
			outputArray[i] = 0
			continue
		}
		outputArray[i] = userInputArray[i]
	}
	outputArray[99] = '\n'
	n, err := existingUserResource.File.WriteAt(outputArray, int64((modifyQuestion)*100))
	if err != nil {
		f.showError(fmt.Sprintf("ERROR in saving the updated user input. %v \n", err), channel)
	}
	fmt.Println(n)
	fmt.Println("Existing user restoring older form")
	lastQuestionAsked, err := f.readAnsAndDisplay(modifyAnsEvent.Channel, existingUserResource.FormName)
	if err != nil {
		f.showError(fmt.Sprintf("Form already exists. Something went wrong in displaying it. %v \n", err), channel)
	}
	existingUserResource.Modify = false
	fmt.Println(lastQuestionAsked)
	if lastQuestionAsked > 3 {
		f.submitForm(modifyAnsEvent, existingUserResource)
		return
	}
	if lastQuestionAsked >= 0 {
		existingUserResource.SyncChannel <- lastQuestionAsked
	}
}

func (f FormBotClient) showError(errorString string, channel string) {
	postMessgeParameters := slack.NewPostMessageParameters()
	postMessgeParameters.Attachments = []slack.Attachment{
		{
			Title: "Following error occurred",
			Text:  errorString,
			Color: "#F35A00",
		},
	}
	f.rtm.PostMessage(channel, "Please contact: Snozzberries team", postMessgeParameters)
}
