package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/azaky/cplinebot/bot"
	"github.com/azaky/cplinebot/clist"

	"github.com/garyburd/redigo/redis"
)

func main() {

	clistService := clist.NewService(os.Getenv("CLIST_APIKEY"), &http.Client{Timeout: 5 * time.Second})

	redisConn, err := redis.Dial("tcp", os.Getenv("REDIS_ENDPOINT"))
	if err != nil {
		log.Fatalf("Error when connecting to redis: %s", err.Error())
	}

	// Setup LineBot
	lineBot := bot.NewLineBot(
		os.Getenv("CHANNEL_SECRET"),
		os.Getenv("CHANNEL_TOKEN"),
		clistService,
		redisConn,
	)
	http.HandleFunc("/line/callback", lineBot.EventHandler)
	lineBot.StartDailyCron(os.Getenv("CRON_SCHEDULE"))

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
