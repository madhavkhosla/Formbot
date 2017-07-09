package main

import (
	"bufio"
	"fmt"
	"os"

	"io/ioutil"

	"github.com/eawsy/aws-lambda-go/service/lambda/runtime"
	"github.com/nlopes/slack"
)

var questions = []string{"q1", "q2", "q3", "q4"}
var Eid string

type Header struct {
	ContentType string `json:"Content-Type"`
}

type SlackResponse struct {
	StatusCode int    `json:"statusCode"`
	Headers    Header `json:"headers"`
	Body       string `json:"body"`
}

type FormBotClient struct {
	rtm        *slack.RTM
	ev         *slack.MessageEvent
	infoUserId string
}

type Set struct {
	Question string
	Answer   string
}

type UserResource struct {
	UserChannel chan int
	UserWriter  *bufio.Writer
}

func init() {
	runtime.HandleFunc(SubmitForm)
}

func main() {
	token := os.Getenv("SLACK_TOKEN")
	api := slack.New(token)
	rtm := api.NewRTM()
	go rtm.ManageConnection()
	userRoutineMap := make(map[string]UserResource)

Loop:
	for {
		msg := <-rtm.IncomingEvents
		fmt.Println("Event Received: ")
		switch ev := msg.Data.(type) {

		case *slack.ConnectedEvent:
			fmt.Println("Connection counter:", ev.ConnectionCount)

		case *slack.MessageEvent:
			fmt.Printf("Message: %v\n", ev.Msg.Text)
			info := rtm.GetInfo()
			formBotClient := FormBotClient{rtm, ev, info.User.ID}

			// Form bot help commands
			formBotClient.helpCommands()

			// Input create command not correct
			check := formBotClient.invalidCreateCommand()
			if !check {
				continue Loop
			}

			existingUserResource, ok := userRoutineMap[ev.User]

			// Form bot start commands
			go formBotClient.startForm(userRoutineMap, ok)

			if ok {
				n3, err := existingUserResource.UserWriter.WriteString(fmt.Sprintf("%s\n", ev.Text))
				if err != nil {
					fmt.Errorf(err.Error())
				}
				fmt.Printf("wrote %s  %d bytes\n", ev.Text, n3)
				existingUserResource.UserWriter.Flush()
				existingUserResource.UserChannel <- 1
			}

		case *slack.RTMError:
			fmt.Printf("Error: %s\n", ev.Error())

		case *slack.InvalidAuthEvent:
			fmt.Printf("Invalid credentials")
			break Loop

		default:
			fmt.Println(msg.Type)
		}

	}
}

func (f FormBotClient) sendQuestions(c chan int, userRoutineMap map[string]UserResource) {
	for _, q := range questions {
		f.rtm.SendMessage(f.rtm.NewOutgoingMessage(fmt.Sprintf("%s", q), f.ev.Channel))
		<-c
	}
	b, err := ioutil.ReadFile(fmt.Sprintf("/Users/madhav/%s", Eid))
	if err != nil {
		fmt.Print(err)
	}
	ansFile := string(b)
	close(c)
	delete(userRoutineMap, f.ev.User)
	fmt.Println(userRoutineMap)
	postMessgeParameters := slack.NewPostMessageParameters()
	postMessgeParameters.AsUser = true
	postMessgeParameters.Attachments = []slack.Attachment{
		{
			Title: "Do you want to submit the intake form",
			Color: "#7CD197",
			Actions: []slack.AttachmentAction{
				{
					Name:  "Submit",
					Text:  "Submit",
					Type:  "button",
					Value: fmt.Sprintf("FS%sFE", ansFile),
				},
			},
			CallbackID: "callbackId",
		},
	}
	f.rtm.PostMessage(f.ev.Channel, "", postMessgeParameters)
}
