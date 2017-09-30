package bot

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/azaky/cpbot/clist"
	"github.com/azaky/cpbot/repository"
	"github.com/azaky/cpbot/util"
	"github.com/line/line-bot-sdk-go/linebot"
)

type LineBot struct {
	clistService *clist.Service
	client       *linebot.Client
	repo         *repository.Redis
	dailyTicker  *time.Ticker
	dailyTimer   map[string]*time.Timer
	dailyNext    time.Time
	dailyPeriod  time.Duration
}

var (
	lineGreetingMessage     = os.Getenv("LINE_GREETING_MESSAGE")
	lineRegexEcho           = regexp.MustCompile(fmt.Sprintf("^@%s\\s+%s\\s*(.*)$", os.Getenv("LINE_BOT_NAME"), "echo"))
	lineRegexShow           = regexp.MustCompile(fmt.Sprintf("^@%s\\s+%s\\s*(.*)$", os.Getenv("LINE_BOT_NAME"), "in"))
	lineRegexDaily          = regexp.MustCompile(fmt.Sprintf("^@%s\\s+%s\\s*([0-9:]*)\\s*$", os.Getenv("LINE_BOT_NAME"), "daily"))
	lineRegexDailyOff       = regexp.MustCompile(fmt.Sprintf("^@%s\\s+%s\\s*%s\\s*$", os.Getenv("LINE_BOT_NAME"), "daily", "off"))
	lineMaxMessageLength, _ = strconv.Atoi(os.Getenv("LINE_MAX_MESSAGE_LENGTH"))
)

func NewLineBot(channelSecret, channelToken string, clistService *clist.Service, redisEndpoint string) *LineBot {
	bot, err := linebot.New(channelSecret, channelToken)
	if err != nil {
		log.Fatalf("Error when initializing linebot: %s", err.Error())
	}
	repo := repository.NewRedis("line", redisEndpoint)
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

func (lb *LineBot) generateGreetingMessage(tz *time.Location) []linebot.Message {
	var messages []linebot.Message
	messages = append(messages, linebot.NewTextMessage(lineGreetingMessage))

	initialReminder, err := generate24HUpcomingContestsMessage(lb.clistService, tz, lineMaxMessageLength)
	if err == nil {
		for _, message := range initialReminder {
			messages = append(messages, linebot.NewTextMessage(message))
		}
	}

	return messages
}

func (lb *LineBot) handleFollow(event linebot.Event) {
	user := util.LineEventSourceToString(event.Source)
	_, err := lb.repo.AddUser(user)
	if err != nil {
		lb.log("Error adding user: %s", err.Error())
	}

	tz, _ := lb.repo.GetTimezone(user)

	messages := lb.generateGreetingMessage(tz)
	if _, err = lb.client.ReplyMessage(event.ReplyToken, messages...).Do(); err != nil {
		lb.log("Error replying to follow event: %s", err.Error())
	}

	// Setup default daily reminder
	t, _ := util.ParseTime(os.Getenv("LINE_DAILY_DEFAULT"))
	lb.updateDaily(user, t)
}

func (lb *LineBot) handleUnfollow(event linebot.Event) {
	user := util.LineEventSourceToString(event.Source)
	_, err := lb.repo.RemoveUser(user)
	if err != nil {
		lb.log("Error removing user: %s", err.Error())
	}
}

func (lb *LineBot) handleTextMessage(event linebot.Event, message *linebot.TextMessage) {
	log.Printf("Received message from %s: %s", event.Source.UserID, message.Text)

	// echo
	if matches := lineRegexEcho.FindStringSubmatch(message.Text); len(matches) > 1 {
		lb.actionEcho(event, matches[1])
		return
	}

	// find contests within duration
	if matches := lineRegexShow.FindStringSubmatch(message.Text); len(matches) > 1 {
		lb.actionShowContestsWithin(event, matches[1])
		return
	}

	// change daily reminder schedule
	if matches := lineRegexDaily.FindStringSubmatch(message.Text); len(matches) > 1 {
		lb.actionUpdateDaily(event, matches[1])
		return
	}

	// turn off daily reminder schedule
	if matches := lineRegexDailyOff.FindStringSubmatch(message.Text); len(matches) > 0 {
		lb.actionRemoveDaily(event)
		return
	}
}

func (lb *LineBot) actionEcho(event linebot.Event, message string) {
	if _, err := lb.client.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message)).Do(); err != nil {
		lb.log("Error replying: %s", err.Error())
	}
}

func (lb *LineBot) actionShowContestsWithin(event linebot.Event, durationStr string) {
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		// Duration is not valid
		reply := fmt.Sprintf("%s is not a valid duration", durationStr)
		if _, err = lb.client.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(reply)).Do(); err != nil {
			lb.log("Error replying: %s", err.Error())
		}
		return
	}

	user := util.LineEventSourceToString(event.Source)
	tz, _ := lb.repo.GetTimezone(user)

	replies, err := generateUpcomingContestsMessage(lb.clistService, time.Now(), time.Now().Add(duration), tz, fmt.Sprintf("Contests starting within %s:", duration), lineMaxMessageLength)
	if err != nil {
		lb.log("Error getting contests: %s", err.Error())
		return
	}

	for _, reply := range replies {
		if _, err = lb.client.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(reply)).Do(); err != nil {
			lb.log("Error replying: %s", err.Error())
		}
	}
}

func (lb *LineBot) actionUpdateDaily(event linebot.Event, tstr string) {
	user := util.LineEventSourceToString(event.Source)
	tz, _ := lb.repo.GetTimezone(user)

	t, err := util.ParseTimeInLocation(tstr, tz)
	if err != nil {
		reply := fmt.Sprintf("%s is not a valid time", tstr)
		if _, err = lb.client.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(reply)).Do(); err != nil {
			lb.log("Error replying: %s", err.Error())
		}
		return
	}

	lb.updateDaily(user, t)
	reply := fmt.Sprintf("Daily contest reminder has been set everyday at %s", tstr)
	if _, err = lb.client.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(reply)).Do(); err != nil {
		lb.log("Error replying: %s", err.Error())
	}
}

func (lb *LineBot) actionRemoveDaily(event linebot.Event) {
	user := util.LineEventSourceToString(event.Source)
	lb.removeDaily(user)
	reply := "Daily contest reminder has been turned off"
	if _, err := lb.client.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(reply)).Do(); err != nil {
		lb.log("Error replying: %s", err.Error())
	}
}

func (lb *LineBot) StartDailyJob(duration time.Duration) {
	if lb.dailyTicker != nil {
		lb.log("An attempt to start daily job, but the job has already started")
		return
	}
	lb.dailyPeriod = duration
	lb.dailyTicker = time.NewTicker(lb.dailyPeriod)

	lb.dailyJob(time.Now())
	go func() {
		for t := range lb.dailyTicker.C {
			lb.dailyJob(t)
		}
	}()
}

func (lb *LineBot) dailyJob(now time.Time) {
	lb.log("[DAILY] Start job")
	lb.dailyNext = now.Add(lb.dailyPeriod)

	userTimes, err := lb.repo.GetDailyWithin(now, lb.dailyNext)
	if err != nil {
		lb.log("[DAILY] Error getting daily within: %s", err.Error())
		return
	}

	lb.log("[DAILY] Schedule for the following users: %v", userTimes)

	lb.dailyTimer = make(map[string]*time.Timer)
	for _, userTime := range userTimes {
		tz, _ := lb.repo.GetTimezone(userTime.User)
		next := util.NextTime(userTime.Time)
		lb.dailyTimer[userTime.User] = time.AfterFunc(next.Sub(time.Now()), lb.dailyReminderFunc(userTime.User, tz))
	}
}

func (lb *LineBot) dailyStarted() bool {
	return lb.dailyTicker != nil
}

func (lb *LineBot) updateDaily(user string, t int) {
	tz, _ := lb.repo.GetTimezone(user)

	_, err := lb.repo.AddDaily(user, t)
	if err != nil {
		lb.log("[DAILY] Error adding to repo (%s, %d): %s", user, t, err.Error())
	}

	if !lb.dailyStarted() {
		return
	}

	if t, ok := lb.dailyTimer[user]; ok {
		t.Stop()
		delete(lb.dailyTimer, user)
	}

	next := util.NextTime(t)
	if next.Before(lb.dailyNext) {
		lb.dailyTimer[user] = time.AfterFunc(next.Sub(time.Now()), lb.dailyReminderFunc(user, tz))
	}
}

func (lb *LineBot) removeDaily(user string) {
	_, err := lb.repo.RemoveDaily(user)
	if err != nil {
		lb.log("[DAILY] Error removing from repo (%s): %s", user, err.Error())
	}

	if !lb.dailyStarted() {
		return
	}

	if t, ok := lb.dailyTimer[user]; ok {
		t.Stop()
		delete(lb.dailyTimer, user)
	}
}

func (lb *LineBot) dailyReminderFunc(user string, tz *time.Location) func() {
	return func() {
		messages, err := generate24HUpcomingContestsMessage(lb.clistService, tz, lineMaxMessageLength)
		if err != nil {
			// TODO: retry mechanism
			lb.log("[DAILY] Error generating message: %s", err.Error())
			return
		}

		eventSource, err := util.StringToLineEventSource(user)
		if err != nil {
			lb.log("[DAILY] found invalid user [%s]: %s", user, err.Error())
			return
		}
		to := fmt.Sprintf("%s", util.LineEventSourceToReplyString(eventSource))
		for _, message := range messages {
			if _, err = lb.client.PushMessage(to, linebot.NewTextMessage(message)).Do(); err != nil {
				lb.log("[CRON] Error sending message to [%s]: %s", to, err.Error())
			}
		}
	}
}
