package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"strings"

	"fmt"
	"net/url"

	"errors"

	"strconv"

	"crypto/aes"
	"crypto/cipher"

	bluele "github.com/bluele/slack"
	"github.com/eawsy/aws-lambda-go/service/lambda/runtime"
	"github.com/nlopes/slack"
)

var key = []byte("123456789012345678901234")

type InteractiveMessageRequest struct {
	Actions []slack.AttachmentAction
	Channel slack.Channel
	User    slack.User
}

func SubmitForm(evt json.RawMessage, ctx *runtime.Context) (interface{}, error) {
	var user map[string]string

	api := bluele.New("xoxb-207019286820-sRB8Ypxgj7MZWscKnVTAye47")

	json.Unmarshal(evt, &user)

	i := strings.Index(user["body"], "=")
	newStr := user["body"][i+1:]
	str, err := url.QueryUnescape(newStr)
	if err != nil {
		log.Printf("Error while decoding the post request %s \n", err.Error())
	}
	log.Printf(str)
	interactiveRequestMessage := InteractiveMessageRequest{}
	err = json.Unmarshal([]byte(str), &interactiveRequestMessage)
	if err != nil {
		log.Printf("Error while un-marshaling request %s \n", err.Error())
	}
	attachmentAction := interactiveRequestMessage.Actions[0]
	if attachmentAction.Name == "Submit" {
		answers := strings.Split(attachmentAction.Value, "$")
		showOutput := make([]Set, 0, len(questions))
		showOutputAttachement := make([]*bluele.Attachment, 0, len(questions))
		for i := 0; i < len(questions); i++ {
			showOutputAttachement = append(showOutputAttachement, &bluele.Attachment{
				Title: questions[i],
				Text:  answers[i],
			})
			showOutput = append(showOutput, Set{Question: questions[i], Answer: answers[i]})
		}

		channel := interactiveRequestMessage.Channel.ID
		userName := interactiveRequestMessage.User.ID
		chatPostOpts := bluele.ChatPostMessageOpt{
			AsUser:      true,
			Attachments: showOutputAttachement,
		}
		err = api.ChatPostMessage(channel, fmt.Sprintf("<@%s> Submitted Form", userName), &chatPostOpts)
		if err != nil {
			log.Printf("Error while posting to slack chat %s\n", err.Error())
		}
		s := SlackResponse{StatusCode: 200,
			Headers: Header{ContentType: "application/json"},
			Body:    ""}

		return s, nil
	} else if attachmentAction.Name == "Select" {

		questionToModify := attachmentAction.SelectedOptions[0].Value
		questionNumber, err := strconv.Atoi(questionToModify)
		if err != nil {
			log.Println(err.Error())
		}
		channel := interactiveRequestMessage.Channel.ID
		userName := interactiveRequestMessage.User.ID
		chatPostOpts := &bluele.ChatPostMessageOpt{
			AsUser: true,
		}

		err = api.ChatPostMessage(channel, fmt.Sprintf("<@%s> Modify Question %d", userName, questionNumber+1), chatPostOpts)
		if err != nil {
			log.Printf("Error while posting to slack chat %s\n", err.Error())
		}
		s := SlackResponse{StatusCode: 200,
			Headers: Header{ContentType: "application/json"},
			Body:    ""}

		return s, nil

	}
	return nil, errors.New("Button selected not correct")
}

// decrypt from base64 to decrypted string
func decrypt(key []byte, cryptoText string) string {
	ciphertext, _ := base64.URLEncoding.DecodeString(cryptoText)

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphertext, ciphertext)

	return fmt.Sprintf("%s", ciphertext)
}
