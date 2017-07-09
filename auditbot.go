package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
	rtm            *slack.RTM
	ev             *slack.MessageEvent
	userRoutineMap map[string]chan int
}

type Set struct {
	Question string
	Answer   string
}

func init() {
	runtime.HandleFunc(OAuth)
}

func OAuth(evt json.RawMessage, ctx *runtime.Context) (interface{}, error) {
	var user map[string]string

	json.Unmarshal(evt, &user)
	text := GetStringInBetween(user["body"], "FS", "FE")
	answers := strings.Split(text, "+")
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

func main() {
	token := os.Getenv("SLACK_TOKEN")
	api := slack.New(token)
	rtm := api.NewRTM()
	go rtm.ManageConnection()
	userRoutineMap := make(map[string]chan int)
	//c := make(chan int)
	//questionCount := 1
	var w *bufio.Writer

Loop:
	for {
		msg := <-rtm.IncomingEvents
		fmt.Println("Event Received: ")
		switch ev := msg.Data.(type) {

		case *slack.ConnectedEvent:
			fmt.Println("Connection counter:", ev.ConnectionCount)

		case *slack.MessageEvent:
			formBotClient := FormBotClient{rtm, ev, userRoutineMap}
			fmt.Printf("Message: %v\n", ev.Msg.Text)
			info := rtm.GetInfo()
			prefix := fmt.Sprintf("<@%s>", info.User.ID)

			// Form bot help commands
			if ev.User != info.User.ID && (ev.Text == prefix || ev.Text == fmt.Sprintf("%s help", prefix)) {
				postMessgeParameters := slack.NewPostMessageParameters()
				postMessgeParameters.Attachments = []slack.Attachment{
					{
						Title: "Command to start new intake form",
						Text:  "@formbot create [EID]",
						Color: "#7CD197",
					},
				}
				rtm.PostMessage(ev.Channel, fmt.Sprintf("Formbot help commands"), postMessgeParameters)
			}
			if ev.User != info.User.ID && strings.HasPrefix(ev.Text, fmt.Sprintf("%s create", prefix)) {
				inputStringLength := strings.Split(ev.Text, " ")

				// Input command not correct
				if len(inputStringLength) != 3 {
					rtm.SendMessage(rtm.NewOutgoingMessage(fmt.Sprintf("Invalid input command"), ev.Channel))
					continue Loop
				}
			}

			existingUserChan, ok := userRoutineMap[ev.User]
			// Form bot start commands
			if ev.User != info.User.ID && strings.HasPrefix(ev.Text, fmt.Sprintf("%s create", prefix)) && !ok {
				inputStringLength := strings.Split(ev.Text, " ")
				Eid = inputStringLength[2]
				trigger := make(chan int)
				userRoutineMap[ev.User] = trigger
				fmt.Println(userRoutineMap)
				go func(trigger chan int) {
					if _, err := os.Stat(fmt.Sprintf("/Users/madhav/%s", Eid)); err != nil {
						if os.IsNotExist(err) {
							// file does not exist
							f, err := os.Create(fmt.Sprintf("/Users/madhav/%s", Eid))
							if err != nil {
								fmt.Errorf("ERROR in creating a file \n")
							}
							w = bufio.NewWriter(f)
							go formBotClient.sendQuestions(trigger)
						} else {
							// other error
						}
					}
				}(trigger)
			}

			if ok {
				n3, err := w.WriteString(fmt.Sprintf("%s\n", ev.Text))
				if err != nil {
					fmt.Errorf(err.Error())
				}
				fmt.Printf("wrote %s  %d bytes\n", ev.Text, n3)
				w.Flush()
				existingUserChan <- 1
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

func (f FormBotClient) sendQuestions(c chan int) {
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
	delete(f.userRoutineMap, f.ev.User)
	fmt.Println(f.userRoutineMap)
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
