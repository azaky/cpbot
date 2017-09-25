package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/azaky/cplinebot/clist"

	"github.com/line/line-bot-sdk-go/linebot"
)

func main() {
	bot, err := linebot.New(
		os.Getenv("CHANNEL_SECRET"),
		os.Getenv("CHANNEL_TOKEN"),
	)
	if err != nil {
		log.Fatalf("Error when initializing linebot: %s", err.Error())
	}

	clistService := clist.NewService(os.Getenv("CLIST_APIKEY"), http.DefaultClient)

	// Setup HTTP Server for receiving requests from LINE platform
	http.HandleFunc("/callback", func(w http.ResponseWriter, req *http.Request) {
		events, err := bot.ParseRequest(req)
		if err != nil {
			if err == linebot.ErrInvalidSignature {
				w.WriteHeader(400)
			} else {
				w.WriteHeader(500)
			}
			return
		}
		for _, event := range events {
			if event.Type == linebot.EventTypeMessage {
				switch message := event.Message.(type) {
				case *linebot.TextMessage:
					log.Printf("Received message from %s", event.Source.UserID)
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message.Text)).Do(); err != nil {
						log.Print(err)
					}
				}
			}
		}
	})

	// Setup Push Message
	http.HandleFunc("/push", func(w http.ResponseWriter, req *http.Request) {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Printf("/push error: %s", err.Error())
			w.WriteHeader(500)
			return
		}
		var reqmap map[string]string
		err = json.Unmarshal(body, &reqmap)
		if err != nil {
			log.Printf("/push error: %s", err.Error())
			w.WriteHeader(500)
			return
		}

		if _, err = bot.PushMessage(reqmap["user"], linebot.NewTextMessage(reqmap["text"])).Do(); err != nil {
			log.Printf("/push error: %s", err.Error())
		}
	})

	// Remind a user about contests in the next 24 hours
	http.HandleFunc("/remind", func(w http.ResponseWriter, req *http.Request) {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Printf("/remind error: %s", err.Error())
			w.WriteHeader(500)
			return
		}
		var reqmap map[string]string
		err = json.Unmarshal(body, &reqmap)
		if err != nil {
			log.Printf("/remind error: %s", err.Error())
			w.WriteHeader(500)
			return
		}

		startFrom := time.Now()
		startEnd := time.Now().Add(86400 * time.Second)
		contests, err := clistService.GetContestsStartingBetween(startFrom, startEnd)
		if err != nil {
			log.Printf("/remind error: %s", err.Error())
			w.WriteHeader(500)
			return
		}

		var buffer bytes.Buffer
		buffer.WriteString("Contests in the next 24 hours:\n")
		for _, contest := range contests {
			buffer.WriteString(fmt.Sprintf("- %s. Starts at %s. Link: %s\n", contest.Name, contest.StartDate.Format("Jan 2 15:04 MST"), contest.Link))
		}

		user := reqmap["user"]
		message := buffer.String()
		log.Println(message)

		if _, err = bot.PushMessage(user, linebot.NewTextMessage(message)).Do(); err != nil {
			log.Printf("/remind error: %s", err.Error())
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"message":"Hello from cplinebot"}`))
	})

	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Fatal(err)
	}
}
