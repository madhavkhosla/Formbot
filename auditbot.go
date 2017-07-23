package main

import (
	"bufio"
	"fmt"
	"os"

	"io/ioutil"

	"strings"

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
	rtm *slack.RTM
	////ev         *slack.MessageEvent
	//infoUserId string
}

type Set struct {
	Question string
	Answer   string
}

type UserResource struct {
	UserChannel chan *slack.MessageEvent
	SyncChannel chan int
	Writer      *bufio.Writer
	QuitChannel chan int
}

func init() {
	runtime.HandleFunc(SubmitForm)
}

func main() {
	token := os.Getenv("SLACK_TOKEN")
	api := slack.New(token)
	rtm := api.NewRTM()
	go rtm.ManageConnection()
	userRoutineMap := make(map[string]*UserResource)
	userFullMap := make(map[string]map[string]*UserResource)
	formBotClient := FormBotClient{rtm}
Loop:
	for {
		msg := <-rtm.IncomingEvents
		fmt.Println("Event Received: ")
		switch ev := msg.Data.(type) {

		case *slack.ConnectedEvent:
			fmt.Println("Connection counter:", ev.ConnectionCount)

		case *slack.MessageEvent:
			fmt.Printf("Message: %v\n", ev.Msg.Text)

			// Form bot help commands
			formBotClient.helpCommands(ev)

			prefix := fmt.Sprintf("<@%s>", formBotClient.rtm.GetInfo().User.ID)
			fmt.Printf("%s-%s\n", ev.User, formBotClient.rtm.GetInfo().User.ID)

			if ev.User != formBotClient.rtm.GetInfo().User.ID && strings.HasPrefix(ev.Text, fmt.Sprintf("%s create", prefix)) {
				// Input create command not correct
				check := formBotClient.invalidCreateCommand(ev)
				if !check {
					continue Loop
				}
				go formBotClient.startForm(ev, userFullMap, userRoutineMap)
			} else if ev.User != formBotClient.rtm.GetInfo().User.ID && len(ev.User) > 0 {
				if len(ev.Text) > 100 {
					rtm.SendMessage(rtm.NewOutgoingMessage(
						fmt.Sprintf("Input should be less than 100 chars. Try Again"), ev.Channel))
					continue
				}
				existingUserResource := userRoutineMap[ev.User]
				existingUserResource.UserChannel <- ev
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

func (f FormBotClient) sendQuestions(ev *slack.MessageEvent, c chan int,
	userFullMap map[string]map[string]*UserResource, startI int, Eid string) {

	for i := startI; i < len(questions); {
		f.rtm.SendMessage(f.rtm.NewOutgoingMessage(fmt.Sprintf("%s", questions[i]), ev.Channel))
		index := <-c
		if index == -1 {
			i = i + 1
		} else {
			fmt.Printf("Index is %v", index)
			i = index
		}
	}
	b, err := ioutil.ReadFile(fmt.Sprintf("/Users/madhav/%s", Eid))
	if err != nil {
		fmt.Print(err)
	}
	ansFile := string(b)
	existingUserResource := userFullMap[ev.User]
	existingUserResource[Eid].QuitChannel <- 0
	close(c)
	close(existingUserResource[Eid].UserChannel)
	close(existingUserResource[Eid].QuitChannel)
	delete(existingUserResource, Eid)
	// Delete File part is left
	fmt.Println(userFullMap)
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
	f.rtm.PostMessage(ev.Channel, "", postMessgeParameters)
}
