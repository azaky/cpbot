package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/azaky/cplinebot/cache"
	"github.com/azaky/cplinebot/clist"

	"github.com/garyburd/redigo/redis"
	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/robfig/cron"
)

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
	messages = append(messages, linebot.NewTextMessage(os.Getenv("GREETING_MESSAGE")))

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

			case linebot.EventTypeLeave:
				fallthrough
			case linebot.EventTypeUnfollow:
				_, err := cacheService.RemoveUser(event.Source)
				if err != nil {
					log.Printf("Error RemoveUser: %s", err.Error())
				}

			case linebot.EventTypeMessage:
				switch message := event.Message.(type) {
				case *linebot.TextMessage:
					log.Printf("Received message from %s: %s", event.Source.UserID, message.Text)
					// echo if it contains @cp-bot
					if strings.Contains(message.Text, "@cp-bot") {
						if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message.Text)).Do(); err != nil {
							log.Printf("Error replying to EventTypeMessage: %s", err.Error())
						}
					}
				}
			}
		}
	})

	// Setup cron job for daily reminder
	job := cron.New()
	job.AddFunc(os.Getenv("CRON_SCHEDULE"), func() {
		log.Printf("[CRON] Start reminder")
		message, err := generate24HUpcomingContestsMessage(clistService)
		if err != nil {
			// TODO: retry mechanism
			log.Printf("[CRON] Error generating message: %s", err.Error())
			return
		}

		users, err := cacheService.GetUsers()
		if err != nil {
			// TODO: retry mechanism
			log.Printf("[CRON] Error getting users: %s", err.Error())
			return
		}

		for _, user := range users {
			userID := fmt.Sprintf("%s%s%s", user.GroupID, user.RoomID, user.UserID)
			if _, err = bot.PushMessage(userID, linebot.NewTextMessage(message)).Do(); err != nil {
				log.Printf("[CRON] Error sending message to [%s]: %s", userID, err.Error())
			}
		}
	})
	job.Start()

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"message":"Hello from cplinebot"}`))
	})

	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Fatal(err)
	}
}
