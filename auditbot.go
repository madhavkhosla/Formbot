package main

import (
	"fmt"
	"os"
	"strings"

	"time"

	"github.com/nlopes/slack"
)

func main() {

	token := os.Getenv("SLACK_TOKEN")
	api := slack.New(token)
	rtm := api.NewRTM()
	go rtm.ManageConnection()

	questions := make([]string, 0, 10)
	answers := make([]string, 0, 10)
	questions = append(questions, "q1", "q2", "q3", "q4")
	questionCount := 1
	c := make(chan int)
	formBotActive := false

Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			fmt.Println("Event Received: ")
			switch ev := msg.Data.(type) {

			case *slack.ConnectedEvent:
				fmt.Println("Connection counter:", ev.ConnectionCount)

			case *slack.MessageEvent:
				fmt.Printf("Message: %v\n", ev)
				info := rtm.GetInfo()
				prefix := fmt.Sprintf("<@%s> ", info.User.ID)
				fmt.Printf("USERS ARE %s %s %s\n", prefix, ev.User, ev.Text)

				if formBotActive {
					if strings.HasPrefix(ev.Text, prefix) {
						formBotActive = false
						answers = make([]string, 0, 10)
						questionCount = 1
						close(c)
					}
					answers = append(answers, ev.Text)
					c <- questionCount
					questionCount++
				}

				if ev.User != info.User.ID && strings.HasPrefix(ev.Text, prefix) {
					formBotActive = true
					go func() {
						for _, question := range questions {
							rtm.SendMessage(rtm.NewOutgoingMessage(fmt.Sprintf(" hello world %s", question), ev.Channel))
							count := <-c
							if count == len(questions) {
								rtm.SendMessage(rtm.NewOutgoingMessage(fmt.Sprintf("%s", answers), ev.Channel))
								formBotActive = false
								answers = make([]string, 0, 10)
								questionCount = 1
								close(c)
							}
						}
					}()
				}

			case *slack.RTMError:
				fmt.Printf("Error: %s\n", ev.Error())

			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials")
				break Loop

			default:
				fmt.Println(msg.Type)
			}

		case <-time.After(60 * time.Second):
			fmt.Printf("Time out \n")
			formBotActive = false
			answers = make([]string, 0, 10)
			questionCount = 1
			close(c)
		}
	}
}
