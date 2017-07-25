package main

import (
	"fmt"
	"strings"

	//"bufio"
	"os"

	"github.com/nlopes/slack"
	//"google.golang.org/appengine/file"
	"strconv"
)

func (formBotClient FormBotClient) startUserRoutine(existingUserResource *UserResource) {
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
				fmt.Errorf(err.Error())
			}

			fmt.Println(existingUserResource)
			fmt.Printf("wrote %s  %d bytes\n", userEvent.Text, n3)
			//existingUserResource.Writer.Flush()
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
	Eid = inputStringLength[2]
	existingUserResource, ok := userFullMap[ev.User][Eid]
	if !ok {
		innerMap := make(map[string]*UserResource)
		// File does not exist.
		// 1) Check if file exists or a new file
		if _, err := os.Stat(fmt.Sprintf("/Users/madhav/%s", Eid)); err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Inside File does not exists")
				// file does not exist
				file, err := os.Create(fmt.Sprintf("/Users/madhav/%s", Eid))
				if err != nil {
					fmt.Errorf("ERROR in creating a file \n")
				}
				//w := bufio.NewWriter(file)
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
			lastAnsSaved, err := formBotClient.readAnsAndDisplay(ev.Channel)
			if err != nil {
				fmt.Errorf("Something went wrong when file already exists and starting form")
			}
			file, err := os.OpenFile(fmt.Sprintf("/Users/madhav/%s", Eid), os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				fmt.Errorf("ERROR in opening a file \n")
			}
			//w := bufio.NewWriter(file)
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
		go formBotClient.startUserRoutine(userRoutineMap[ev.User])
	} else {
		fmt.Println("Existing user restoring older form")
		lastQuestionAsked, err := formBotClient.readAnsAndDisplay(ev.Channel)
		if err != nil {
			fmt.Printf("Something went wrong")
		}
		// If someone calls modify and then switches forms, when he comes back to first form
		// Modify flag is removed.
		existingUserResource.Modify = false
		userRoutineMap[ev.User] = existingUserResource
		if lastQuestionAsked >= 0 {
			userRoutineMap[ev.User].SyncChannel <- lastQuestionAsked
		}
	}

}

func (f FormBotClient) invalidCreateCommand(ev *slack.MessageEvent) bool {
	inputStringLength := strings.Split(ev.Text, " ")
	if len(inputStringLength) != 3 {
		f.rtm.SendMessage(f.rtm.NewOutgoingMessage(fmt.Sprintf("Invalid input command"), ev.Channel))
		return false
	}
	return true
}

func (f FormBotClient) helpCommands(ev *slack.MessageEvent) {
	botId := f.rtm.GetInfo().User.ID
	prefix := fmt.Sprintf("<@%s>", botId)
	if ev.User != botId && (ev.Text == prefix || ev.Text == fmt.Sprintf("%s help", prefix)) {
		postMessgeParameters := slack.NewPostMessageParameters()
		postMessgeParameters.Attachments = []slack.Attachment{
			{
				Title: "Command to start new intake form",
				Text:  "@formbot create [EID]",
				Color: "#7CD197",
			},
		}
		f.rtm.PostMessage(ev.Channel, fmt.Sprintf("Formbot help commands"), postMessgeParameters)
	}
}

func (f FormBotClient) updateAnswer(ev *slack.MessageEvent, existingUserResource *UserResource) {
	inputStringLength := strings.Split(ev.Text, " ")
	modifyQuestionString := inputStringLength[3]
	modifyQuestion, err := strconv.Atoi(modifyQuestionString)
	if err != nil {
		fmt.Errorf("Error in update Answer in converting question number to int")
	}
	f.rtm.SendMessage(f.rtm.NewOutgoingMessage(fmt.Sprintf("%s", questions[modifyQuestion]), ev.Channel))
	go f.modifyAnswerRoutine(modifyQuestion, existingUserResource)
}

func (f FormBotClient) modifyAnswerRoutine(modifyQuestion int, existingUserResource *UserResource) {

	modifyAnsEvent := <-existingUserResource.ModifyChannel

	userInputArray := []byte(modifyAnsEvent.Text)
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
		fmt.Errorf("%s", err.Error())
	}
	fmt.Println(n)
	fmt.Println("Existing user restoring older form")
	lastQuestionAsked, err := f.readAnsAndDisplay(modifyAnsEvent.Channel)
	if err != nil {
		fmt.Printf("Something went wrong")
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
