package main

import (
	"log"
	"os"
	"time"

	"home-alarm-bot/internal/httpapi"
	"home-alarm-bot/internal/state"
	"home-alarm-bot/internal/telegram"
	"home-alarm-bot/internal/alarm"

	"github.com/joho/godotenv"
)

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("missing %s", k)
	}
	return v
}

func main() {
	_ = godotenv.Load()

	alarmClient := alarm.New(mustEnv("SERVER_BASE_URL"))
	tgAPI := telegram.NewAPI(mustEnv("BOT_TOKEN"))
	store := state.New()
	bot   := telegram.NewBot(tgAPI, store, alarmClient)

	// start local HTTP listener in a goroutine
	go func() {
		srv := httpapi.New(store, bot)
		if err := srv.Listen("127.0.0.1:8080"); err != nil {
			log.Fatal(err)
		}
	}()

	// Telegram long-poll loop
	var offset int
	for {
		updates, err := tgAPI.GetUpdates(offset)
		if err != nil {
			log.Println("getUpdates:", err)
			time.Sleep(5 * time.Second)
			continue
		}
		for _, u := range updates {
			bot.Handle(u)
			if u.UpdateID >= offset {
				offset = u.UpdateID + 1
			}
		}
	}
}
