package main

import (
	"fmt"
	"os"
	"strings"

	"bufio"

	"github.com/nlopes/slack"
)

var questions = []string{"q1", "q2", "q3", "q4"}
var formbot bool = false

type FormBotClient struct {
	rtm *slack.RTM
	ev  *slack.MessageEvent
}

func main() {

	token := os.Getenv("SLACK_TOKEN")
	api := slack.New(token)
	rtm := api.NewRTM()
	go rtm.ManageConnection()
	c := make(chan int)
	questionCount := 1
	var w *bufio.Writer

Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			fmt.Println("Event Received: ")
			switch ev := msg.Data.(type) {

			case *slack.ConnectedEvent:
				fmt.Println("Connection counter:", ev.ConnectionCount)

			case *slack.MessageEvent:
				formBotClient := FormBotClient{rtm, ev}
				fmt.Printf("Message: %v\n", ev.Msg.Text)
				info := rtm.GetInfo()
				prefix := fmt.Sprintf("<@%s>", info.User.ID)

				// Audit bot help commands
				if ev.User != info.User.ID && (ev.Text == prefix || ev.Text == fmt.Sprintf("%s help", prefix)) {
					postMessgeParameters := slack.NewPostMessageParameters()
					postMessgeParameters.Attachments = []slack.Attachment{
						{
							Title: "Command to start new intake form",
							Text:  "@auditbot create [EID]",
							Color: "#7CD197",
						},
					}
					rtm.PostMessage(ev.Channel, fmt.Sprintf("Auditbot help commands"), postMessgeParameters)
				}

				// Audit bot start commands
				if ev.User != info.User.ID && strings.HasPrefix(ev.Text, fmt.Sprintf("%s create", prefix)) {
					inputStringLength := strings.Split(ev.Text, " ")

					// Input command not correct
					if len(inputStringLength) != 3 {
						rtm.SendMessage(rtm.NewOutgoingMessage(fmt.Sprintf("Invalid input command"), ev.Channel))
					} else {
						if _, err := os.Stat(fmt.Sprintf("/Users/madhav/%s", inputStringLength[2])); err != nil {
							if os.IsNotExist(err) {
								// file does not exist
								f, err := os.Create(fmt.Sprintf("/Users/madhav/%s", inputStringLength[2]))
								if err != nil {
									fmt.Errorf("ERROR in creating a file \n")
								}
								w = bufio.NewWriter(f)
								go formBotClient.sendQuestions(c)
							} else {
								// other error
							}
						}
					}
				}
				if formbot {
					n3, err := w.WriteString(fmt.Sprintf("%s\n", ev.Text))
					if err != nil {
						fmt.Errorf(err.Error())
					}
					fmt.Printf("wrote %s  %d bytes\n", ev.Text, n3)
					w.Flush()
					c <- questionCount
					questionCount++
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
}

func (f FormBotClient) sendQuestions(c chan int) {
	formbot = true
	for _, q := range questions {
		f.rtm.SendMessage(f.rtm.NewOutgoingMessage(fmt.Sprintf("%s", q), f.ev.Channel))
		questionCount := <-c
		if questionCount == len(questions) {
			formbot = false
			close(c)
			postMessgeParameters := slack.NewPostMessageParameters()
			postMessgeParameters.Attachments = []slack.Attachment{
				{
					Title: "Do you want to submit the intake form",
					Color: "#7CD197",
					Actions: []slack.AttachmentAction{
						{
							Name:  "Submit",
							Text:  "Submit",
							Type:  "button",
							Value: "submit",
						},
					},
				},
			}
			f.rtm.PostMessage(f.ev.Channel, "", postMessgeParameters)
		}
	}
}
