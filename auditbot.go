package main

import (
	"fmt"
	"os"

	"strings"

	"io/ioutil"

	"strconv"

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
}

type Set struct {
	Question string
	Answer   string
}

type UserResource struct {
	UserChannel   chan *slack.MessageEvent
	ModifyChannel chan *slack.MessageEvent
	SyncChannel   chan int
	File          *os.File
	QuitChannel   chan int
	FormName      string
	Modify        bool
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
			fmt.Println(ev.Msg.BotID)
			fmt.Printf("Message: %v\n", ev.Msg.Text)

			// Form bot help commands
			helpCommandInvoked := formBotClient.helpCommands(ev)
			if helpCommandInvoked {
				continue
			}

			prefix := fmt.Sprintf("<@%s>", formBotClient.rtm.GetInfo().User.ID)

			if ev.User != formBotClient.rtm.GetInfo().User.ID && strings.HasPrefix(ev.Text, fmt.Sprintf("%s create", prefix)) {
				// Input create command not correct
				check := formBotClient.invalidCreateCommand(ev)
				if !check {
					continue Loop
				}
				// Create command starts a form
				go formBotClient.startForm(ev, userFullMap, userRoutineMap)
			} else if ev.User != formBotClient.rtm.GetInfo().User.ID && ev.Text == fmt.Sprintf("%s modify", prefix) {
				// Modify command invoked by user to modify a particular question
				existingUserResource := userRoutineMap[ev.User]
				if existingUserResource != nil {
					formBotClient.modifyMenu(ev)
				} else {
					rtm.SendMessage(rtm.NewOutgoingMessage(
						fmt.Sprintf("Please start form before modifying answers"), ev.Channel))
					continue
				}
			} else if strings.Contains(ev.Text, "Submitted Form") {
				// This is invoked by the formbot, to submit the form
				inputStringLength := strings.Split(ev.Text, " ")
				user := inputStringLength[0]
				fmt.Println(user)
				existingUserResource := userFullMap[user[2:len(user)-1]]
				existingUserResource[Eid].QuitChannel <- 0
				close(existingUserResource[Eid].SyncChannel)
				close(existingUserResource[Eid].UserChannel)
				close(existingUserResource[Eid].QuitChannel)
				close(existingUserResource[Eid].ModifyChannel)
				delete(existingUserResource, Eid)
				delete(userRoutineMap, user[2:len(user)-1])
				// Delete File part is left
				fmt.Println(userFullMap)
			} else if strings.Contains(ev.Text, "Modify Question") {
				// This is invoked by the formbot, to modify the question specified by user in the above step
				inputStringLength := strings.Split(ev.Text, " ")
				user := inputStringLength[0]
				fmt.Println(user)
				existingUserResource := userRoutineMap[user[2:len(user)-1]]
				existingUserResource.Modify = true
				go formBotClient.updateAnswer(ev, existingUserResource)
			} else if ev.User != formBotClient.rtm.GetInfo().User.ID && len(ev.User) > 0 {
				if len(ev.Text) > 100 {
					rtm.SendMessage(rtm.NewOutgoingMessage(
						fmt.Sprintf("User input should be less than 100 chars. Try Again"), ev.Channel))
					continue
				}
				existingUserResource := userRoutineMap[ev.User]
				if existingUserResource.Modify {
					existingUserResource.ModifyChannel <- ev
					continue
				}
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
			fmt.Printf("Index is %v\n", index)
			i = index
		}
	}
	existingUserResource := userFullMap[ev.User]
	f.submitForm(ev, existingUserResource[Eid])
}

func (f FormBotClient) submitForm(ev *slack.MessageEvent, existingUserResource *UserResource) {
	b, err := ioutil.ReadFile(fmt.Sprintf("/Users/madhav/%s", Eid))
	if err != nil {
		f.showError(fmt.Sprintf("ERROR in Reading saved input form. %v \n", err), ev.Channel)
	}
	ansFile := string(b)
	fmt.Println(fmt.Sprintf("%s", ansFile))
	postMessgeParameters := slack.NewPostMessageParameters()
	postMessgeParameters.AsUser = true
	questionOptions := []slack.AttachmentActionOption{}

	for i, q := range questions {
		questionOptions = append(questionOptions, slack.AttachmentActionOption{Text: q, Value: strconv.Itoa(i)})
	}
	postMessgeParameters.Attachments = []slack.Attachment{
		{
			Title: "Do you want to submit the intake form",
			Color: "#7CD197",
			Actions: []slack.AttachmentAction{
				{
					Name:  "Submit",
					Text:  "Submit",
					Type:  "button",
					Value: fmt.Sprintf("%s", ansFile),
				},
				{
					Name:    "Select",
					Type:    "select",
					Text:    "Modify Question",
					Options: questionOptions,
				},
			},
			CallbackID: "callbackId",
		},
	}
	f.rtm.PostMessage(ev.Channel, "", postMessgeParameters)

}

func (f FormBotClient) modifyMenu(ev *slack.MessageEvent) {

	questionOptions := []slack.AttachmentActionOption{}

	for i, q := range questions {
		questionOptions = append(questionOptions, slack.AttachmentActionOption{Text: q, Value: strconv.Itoa(i)})
	}

	attachment := slack.Attachment{
		Text:       "Choose the question to modify",
		Color:      "#7CD197",
		CallbackID: "menuCallbackId",
		Actions: []slack.AttachmentAction{
			{
				Name:    "Select",
				Type:    "select",
				Text:    "Modify Question",
				Options: questionOptions,
			},
		},
	}

	params := slack.PostMessageParameters{
		Attachments: []slack.Attachment{
			attachment,
		},
		AsUser: true,
	}

	if _, _, err := f.rtm.PostMessage(ev.Channel, "", params); err != nil {
		f.showError(fmt.Sprintf("ERROR in Posting message to slack. %v \n", err), ev.Channel)
	}
}
