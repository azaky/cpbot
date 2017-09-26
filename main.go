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

	"github.com/azaky/cplinebot/cache"
	"github.com/azaky/cplinebot/clist"

	"github.com/garyburd/redigo/redis"
	"github.com/line/line-bot-sdk-go/linebot"
)

const greetingMessage = `Thanks for adding me!

I will remind you the schedule of upcoming competitive programming contests. Contest times are provided by this awesome https://clist.by by Aleksey Ropan`

func generate24HUpcomingContestsMessage(clistService clist.Service) (string, error) {
	startFrom := time.Now()
	startTo := time.Now().Add(86400 * time.Second)
	contests, err := clistService.GetContestsStartingBetween(startFrom, startTo)
	if err != nil {
		log.Printf("Error generate24HUpcomingContestsMessage: %s", err.Error())
		return "", err
	}

	var buffer bytes.Buffer
	buffer.WriteString("Contests in the next 24 hours:\n")
	for _, contest := range contests {
		buffer.WriteString(fmt.Sprintf("- %s. Starts at %s. Link: %s\n", contest.Name, contest.StartDate.Format("Jan 2 15:04 MST"), contest.Link))
	}

	return buffer.String(), nil
}

func generateGreetingMessage(clistService clist.Service) []linebot.Message {
	var messages []linebot.Message
	messages = append(messages, linebot.NewTextMessage(greetingMessage))

	initialReminder, err := generate24HUpcomingContestsMessage(clistService)
	if err == nil {
		messages = append(messages, linebot.NewTextMessage(initialReminder))
	}

	return messages
}

func main() {
	bot, err := linebot.New(
		os.Getenv("CHANNEL_SECRET"),
		os.Getenv("CHANNEL_TOKEN"),
	)
	if err != nil {
		log.Fatalf("Error when initializing linebot: %s", err.Error())
	}

	redisConn, err := redis.Dial("tcp", os.Getenv("REDIS_ENDPOINT"))
	if err != nil {
		log.Fatalf("Error when connecting to redis: %s", err.Error())
	}

	clistService := clist.NewService(os.Getenv("CLIST_APIKEY"), &http.Client{Timeout: 5 * time.Second})
	cacheService := cache.NewService(redisConn)

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
			log.Printf("[EVENT][%s] Source: %#v", event.Type, event.Source)
			switch event.Type {
			case linebot.EventTypeJoin:
				fallthrough
			case linebot.EventTypeFollow:
				_, err := cacheService.AddUser(event.Source)
				if err != nil {
					log.Printf("Error AddUser: %s", err.Error())
				}
				messages := generateGreetingMessage(clistService)
				if _, err = bot.ReplyMessage(event.ReplyToken, messages...).Do(); err != nil {
					log.Printf("Error replying to EventTypeJoin: %s", err.Error())
				}
			case linebot.EventTypeMessage:
				switch message := event.Message.(type) {
				case *linebot.TextMessage:
					log.Printf("Received message from %s: %s", event.Source.UserID, message.Text)
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message.Text)).Do(); err != nil {
						log.Printf("Error replying to EventTypeMessage: %s", err.Error())
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
		startTo := time.Now().Add(86400 * time.Second)
		contests, err := clistService.GetContestsStartingBetween(startFrom, startTo)
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
