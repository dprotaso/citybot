package main

import (
	"os"
	"fmt"
	"net/http"
	"errors"

	fbbot "github.com/ippy04/messengerbot"
	cl "github.com/mpmlj/clarifai-client-go"
	"sync"
)


var session *cl.Session
var state map[string]string
var stateMutex sync.RWMutex

func main() {
	state = make(map[string]string)

	var port string
	var verifyToken string
	var accessToken string

	var clientId string
	var clientSecret string

	if port = os.Getenv("PORT"); port == "" {
		port = "3000"
	}

	if verifyToken = os.Getenv("MESSENGER_VERIFY_TOKEN"); verifyToken == "" {
		panic("Environment variable 'MESSENGER_VERIFY_TOKEN' must be set")
	}

	if accessToken = os.Getenv("MESSENGER_ACCESS_TOKEN"); accessToken == "" {
		panic("Environment variable 'MESSENGER_ACCESS_TOKEN' must be set")
	}

	if clientId = os.Getenv("CLARIFAI_CLIENT_ID"); clientId == "" {
		panic("Environment variable 'CLARIFAI_CLIENT_ID' must be set")
	}

	if clientSecret = os.Getenv("CLARIFAI_CLIENT_SECRET"); clientSecret == "" {
		panic("Environment variable 'CLARIFAI_CLIENT_SECRET' must be set")
	}

	var err error
	session, err = cl.Connect(clientId, clientSecret)

	if err != nil {
		panic("Error connecting to clarifai " + err.Error())
	}

	bot := fbbot.NewMessengerBot(accessToken, verifyToken)

	bot.MessageReceived = messageReceivedHandler
	bot.MessageDelivered = messageDeliveredHandler
	bot.Postback = postbackHandler
	bot.Authentication = authenticationHandler

	fmt.Printf("Listening on port :%s\n", port)
	http.HandleFunc("/", bot.Handler)
	http.ListenAndServe(":" + port, nil)
}



func messageReceivedHandler(
	bot *fbbot.MessengerBot,
	event fbbot.Event,
	opts fbbot.MessageOpts,
	msg fbbot.ReceivedMessage) {

	fmt.Printf("Message received from %s - %s\n", opts.Sender.ID, msg.Text)

	if len(msg.Attachments) == 0 {
		sendUsage(bot, opts)
		return
	}

	if msg.Attachments[0].Type == "image" {
		if hasPendingPicture(opts.Sender.ID) {
			clearPendingPicture(opts.Sender.ID)
		}
		handleImageAttachment(bot, opts, msg)
		return
	}

	if hasPendingPicture(opts.Sender.ID) && msg.Attachments[0].Type == "location" {
		clearPendingPicture(opts.Sender.ID)
		sendReply(bot, opts, "We'll send our crews out there! Stay tuned!")
		return
	}

	sendReply(bot, opts, fmt.Sprintf("Thanks for sending me a %s", msg.Attachments[0].Type))
}

func handleImageAttachment(bot *fbbot.MessengerBot, opts fbbot.MessageOpts, msg fbbot.ReceivedMessage) {
	isHole, err := isPotholePicture(msg.Attachments[0])

	if err != nil {
		fmt.Println("error occurred " + err.Error())
		sendReply(bot, opts, "Sorry. I lost my glasses and I'm having trouble looking at the picture.")
		return
	} else if !isHole {
		sendNotStupid(bot, opts)
		return
	}

	sendReply(bot, opts, "Thanks for letting us know about the pot hole - can you send your location?")

	stateMutex.Lock()
	defer stateMutex.Unlock()

	state[opts.Sender.ID] = opts.Sender.ID
}

func clearPendingPicture(id string) {
	stateMutex.Lock()
	defer stateMutex.Unlock()
	delete(state, id)
}

func hasPendingPicture(id string) bool {
	stateMutex.Lock()
	defer stateMutex.Unlock()

	_, ok := state[id]

	return ok
}



func isPotholePicture(attachment *fbbot.Attachment) (bool, error) {
	payload := attachment.Payload.(map[string]interface{})
	url := payload["url"].(string)

	data := cl.InitInputs()
	data.AddInput(cl.NewImageFromURL(url), "")

	data.SetModel("Stuff")

	resp, err := session.Predict(data).Do()

	if err != nil {
		return false, err
	}

	if len(resp.Outputs) == 0 {
		return false, errors.New("No outputs from clarify")
	}

	for _, concept := range resp.Outputs[0].Data.Concepts {
		fmt.Printf("concept %s with confidence %v\n", concept.Name, concept.Value)

		if concept.Name == "pothole" && concept.Value > 0.8 {
			return true, nil
		}

	}

	return false, nil
}

func sendReply(bot *fbbot.MessengerBot, opts fbbot.MessageOpts, msg string) {
	bot.SendTextMessage(
		fbbot.NewUserFromId(opts.Sender.ID),
		msg,
		fbbot.NotificationTypeRegular,
	)
}


func sendNotStupid(bot *fbbot.MessengerBot, opts fbbot.MessageOpts) {
	message := "I'm educated. That is no pothole!"
	profile, err := bot.GetProfile(opts.Sender.ID)

	if err == nil {
		message = fmt.Sprintf("%s - %s", profile.FirstName, message)
	} else {
		fmt.Printf("Error fetching profile %s", err)
	}

	sendReply(bot, opts, message)
}

func sendUsage(bot *fbbot.MessengerBot, opts fbbot.MessageOpts) {
	message := "Send me pictures of pot holes that you find!"
	profile, err := bot.GetProfile(opts.Sender.ID)

	if err == nil {
		message = fmt.Sprintf("Hi %s! %s", profile.FirstName, message)
	} else {
		fmt.Printf("Error fetching profile %s", err)
		message = fmt.Sprintf("Hi! %s", message)
	}

	sendReply(bot, opts, message)
}

func messageDeliveredHandler(
	bot *fbbot.MessengerBot,
	event fbbot.Event,
	opts fbbot.MessageOpts,
	msg fbbot.Delivery) {

}
func postbackHandler(
	bot *fbbot.MessengerBot,
	event fbbot.Event,
	opts fbbot.MessageOpts,
	postback fbbot.Postback) {

}

func authenticationHandler(
	bot *fbbot.MessengerBot,
	event fbbot.Event,
	opts fbbot.MessageOpts,
	optin *fbbot.Optin) {

}
