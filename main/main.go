package main

import (
	"config"
	"log"
	"strings"
	"youtubeHandler"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	cfg := config.LoadConfig("config.json")

	botToken := cfg.TGBotKey
	if botToken == "" {
		log.Fatal("TGBotKey is not set")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal(err)
	}

	bot.Debug = false
	log.Printf("Authorized as @%s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if strings.Contains(update.Message.Text, "youtube.com") ||
			strings.Contains(update.Message.Text, "youtu.be") {
			go youtube.HandleYouTube(bot, update.Message)
		} else {

		}
	}
}
