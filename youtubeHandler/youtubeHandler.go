package youtube

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kkdai/youtube/v2"
)

func HandleYouTube(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID

	bot.Send(tgbotapi.NewMessage(chatID, "Скачиваю видео..."))

	client := youtube.Client{}

	video, err := client.GetVideoContext(context.Background(), msg.Text)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка получения видео"))
		return
	}

	var format *youtube.Format
	for _, f := range video.Formats {
		if strings.HasPrefix(f.MimeType, "video/") && f.AudioChannels > 0 {
			format = &f
			break
		}
	}

	if format == nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Не найден подходящий формат"))
		return
	}

	stream, _, err := client.GetStream(video, format)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка скачивания"))
		return
	}
	defer stream.Close()

	tmpFile, err := os.CreateTemp("/tmp", "yt-*.mp4")
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка создания файла"))
		return
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.ReadFrom(stream); err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка записи"))
		return
	}

	if _, err := tmpFile.Seek(0, 0); err != nil {
		return
	}

	videoMsg := tgbotapi.NewVideo(chatID, tgbotapi.FileReader{
		Name:   fmt.Sprintf("%s.mp4", video.ID),
		Reader: tmpFile,
	})

	videoMsg.Caption = video.Title

	if _, err := bot.Send(videoMsg); err != nil {
		log.Println("send error:", err)
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка отправки видео"))
	}
}
