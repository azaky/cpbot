package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/azaky/cpbot/bot"
	"github.com/azaky/cpbot/clist"
)

func main() {

	clistService := clist.NewService(os.Getenv("CLIST_APIKEY"), &http.Client{Timeout: 5 * time.Second})

	// Setup LineBot
	lineBot := bot.NewLineBot(
		os.Getenv("LINE_CHANNEL_SECRET"),
		os.Getenv("LINE_CHANNEL_TOKEN"),
		clistService,
		os.Getenv("REDIS_ENDPOINT"),
	)
	http.HandleFunc("/line/callback", lineBot.EventHandler)
	lineDailyDuration, err := strconv.ParseInt(os.Getenv("LINE_DAILY_PERIOD"), 10, 64)
	if err != nil {
		lineDailyDuration = 1800
	}
	lineBot.StartDailyJob(time.Duration(lineDailyDuration) * time.Second)

	// Setup root endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"message":"Hello from cpbot"}`))
	})

	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Fatal(err)
	}
}
