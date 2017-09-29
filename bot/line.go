package bot

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/robfig/cron"

	"github.com/azaky/cpbot/clist"
	"github.com/azaky/cpbot/repository"
	"github.com/azaky/cpbot/util"
	"github.com/garyburd/redigo/redis"
	"github.com/line/line-bot-sdk-go/linebot"
)

type LineBot struct {
	clistService *clist.Service
	client       *linebot.Client
	repo         *repository.Redis
	dailyCron    *cron.Cron
}

var (
	lineGreetingMessage = os.Getenv("LINE_GREETING_MESSAGE")
	lineRegexEcho       = regexp.MustCompile(fmt.Sprintf("@%s\\s+%s\\s*(.*)", os.Getenv("LINE_BOT_NAME"), "echo"))
	lineRegexShow       = regexp.MustCompile(fmt.Sprintf("@%s\\s+%s\\s*(.*)", os.Getenv("LINE_BOT_NAME"), "in"))
)

func NewLineBot(channelSecret, channelToken string, clistService *clist.Service, redisConn redis.Conn) *LineBot {
	bot, err := linebot.New(channelSecret, channelToken)
	if err != nil {
		log.Fatalf("Error when initializing linebot: %s", err.Error())
	}
	repo := repository.NewRedis("line", redisConn)
	return &LineBot{
		clistService: clistService,
		client:       bot,
		repo:         repo,
	}
}

func (lb *LineBot) log(format string, args ...interface{}) {
	log.Printf("[LINE] "+format, args...)
}

func (lb *LineBot) EventHandler(w http.ResponseWriter, req *http.Request) {
	events, err := lb.client.ParseRequest(req)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	for _, event := range events {
		lb.log("[EVENT][%s] Source: %#v", event.Type, event.Source)
		switch event.Type {

		case linebot.EventTypeJoin:
			fallthrough
		case linebot.EventTypeFollow:
			lb.handleFollow(event)

		case linebot.EventTypeLeave:
			fallthrough
		case linebot.EventTypeUnfollow:
			lb.handleUnfollow(event)

		case linebot.EventTypeMessage:
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				lb.handleTextMessage(event, message)
			}
		}
	}
}

func (lb *LineBot) StartDailyCron(schedule string) {
	if lb.dailyCron != nil {
		lb.log("An attempt to start daily cron, but the job has already started")
		return
	}
	job := cron.New()
	job.AddFunc(schedule, func() {
		lb.log("[CRON] Start reminder")
		message, err := generate24HUpcomingContestsMessage(lb.clistService)
		if err != nil {
			// TODO: retry mechanism
			lb.log("[CRON] Error generating message: %s", err.Error())
			return
		}

		users, err := lb.repo.GetUsers()
		if err != nil {
			// TODO: retry mechanism
			lb.log("[CRON] Error getting users: %s", err.Error())
			return
		}

		for _, user := range users {
			eventSource, err := util.StringToLineEventSource(user)
			if err != nil {
				lb.log("[CRON] found invalid user [%s]: %s", user, err.Error())
				continue
			}
			to := fmt.Sprintf("%s%s%s", eventSource.GroupID, eventSource.RoomID, eventSource.UserID)
			if _, err = lb.client.PushMessage(to, linebot.NewTextMessage(message)).Do(); err != nil {
				lb.log("[CRON] Error sending message to [%s]: %s", to, err.Error())
			}
		}
	})
	job.Start()
	lb.dailyCron = job
}

func (lb *LineBot) generateGreetingMessage() []linebot.Message {
	var messages []linebot.Message
	messages = append(messages, linebot.NewTextMessage(lineGreetingMessage))

	initialReminder, err := generate24HUpcomingContestsMessage(lb.clistService)
	if err == nil {
		messages = append(messages, linebot.NewTextMessage(initialReminder))
	}

	return messages
}

func (lb *LineBot) handleFollow(event linebot.Event) {
	_, err := lb.repo.AddUser(util.LineEventSourceToString(event.Source))
	if err != nil {
		lb.log("Error adding user: %s", err.Error())
	}
	messages := lb.generateGreetingMessage()
	if _, err = lb.client.ReplyMessage(event.ReplyToken, messages...).Do(); err != nil {
		lb.log("Error replying to follow event: %s", err.Error())
	}
}

func (lb *LineBot) handleUnfollow(event linebot.Event) {
	_, err := lb.repo.RemoveUser(util.LineEventSourceToString(event.Source))
	if err != nil {
		lb.log("Error removing user: %s", err.Error())
	}
}

func (lb *LineBot) handleTextMessage(event linebot.Event, message *linebot.TextMessage) {
	log.Printf("Received message from %s: %s", event.Source.UserID, message.Text)

	// echo
	if matches := lineRegexEcho.FindStringSubmatch(message.Text); len(matches) > 0 {
		lb.echo(event, matches[1])
	}

	// find contests within duration
	if matches := lineRegexShow.FindStringSubmatch(message.Text); len(matches) > 0 {
		lb.showContestsWithin(event, matches[1])
	}
}

func (lb *LineBot) echo(event linebot.Event, message string) {
	if _, err := lb.client.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message)).Do(); err != nil {
		lb.log("Error replying: %s", err.Error())
	}
}

func (lb *LineBot) showContestsWithin(event linebot.Event, durationStr string) {
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		// Duration is not valid
		reply := fmt.Sprintf("%s is not a valid duration", durationStr)
		if _, err = lb.client.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(reply)).Do(); err != nil {
			lb.log("Error replying: %s", err.Error())
		}
		return
	}

	reply, err := generateUpcomingContestsMessage(lb.clistService, time.Now(), time.Now().Add(duration), fmt.Sprintf("Contests starting within %s:", duration))
	if err != nil {
		lb.log("Error getting contests: %s", err.Error())
		return
	}

	if _, err = lb.client.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(reply)).Do(); err != nil {
		lb.log("Error replying: %s", err.Error())
	}
}
