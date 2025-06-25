package main

import (
	"log"
	"os"
	"time"

	"home-alarm-bot/internal/alarm"
	"home-alarm-bot/internal/telegram"

	"github.com/joho/godotenv"
)

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("missing %s,", k)
	}
	return v
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("error loading .env file: %v", err)
	}
	tg := telegram.NewAPI(mustEnv("BOT_TOKEN"))
	pi := alarm.New(mustEnv("SERVER_BASE_URL"))
	bot := telegram.NewBot(tg, pi)

	var offset int

	for {
		updates, err := tg.GetUpdates(offset)
		if err != nil {
			log.Println("getUpdates:", err)
			time.Sleep(5 * time.Second)
		}
		for _, u := range updates {
			bot.Handle(u)
			if u.UpdateID >= offset {
				offset = u.UpdateID + 1
			}
		}
	}
}