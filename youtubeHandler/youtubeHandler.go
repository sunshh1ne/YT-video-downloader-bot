package youtubeHandler

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/kkdai/youtube/v2"
)

type VideoOption struct {
	Quality int
	Format  *youtube.Format
}

func getOptions(video *youtube.Video) []VideoOption {
	var options []VideoOption
	seenQualities := make(map[int]bool)
	for _, f := range video.Formats {
		if strings.HasPrefix(f.MimeType, "video/") {
			if f.Height > 1080 {
				continue
			}
			if !seenQualities[f.Height] {
				options = append(options, VideoOption{
					Quality: f.Height,
					Format:  &f,
				})
				seenQualities[f.Height] = true
			}
		}
	}
	return options
}

func HandleYouTube(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	client := youtube.Client{}

	video, err := client.GetVideoContext(context.Background(), msg.Text)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка получения информации о видео"))
		return
	}

	options := getOptions(video)

	if len(options) == 0 {
		bot.Send(tgbotapi.NewMessage(chatID, "Не удалось найти доступные форматы видео"))
		return
	}

	var inlineKeyboard [][]tgbotapi.InlineKeyboardButton
	for i, opt := range options {
		callbackData := fmt.Sprintf("%s|%s|%d", "sq", msg.Text, i)
		button := tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(opt.Quality)+"p", callbackData)
		inlineKeyboard = append(inlineKeyboard, tgbotapi.NewInlineKeyboardRow(button))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(inlineKeyboard...)
	msg1 := tgbotapi.NewMessage(chatID, "Выберите качество видео:")
	msg1.ReplyMarkup = keyboard
	if _, err := bot.Send(msg1); err != nil {
		log.Printf("Ошибка отправки сообщения с клавиатурой: %v", err)
	}
}

func HandleCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	chatID := callback.Message.Chat.ID
	messageID := callback.Message.MessageID
	data := callback.Data

	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	if _, err := bot.Request(deleteMsg); err != nil {
		log.Printf("Ошибка удаления сообщения: %v", err)
	}

	parts := strings.Split(data, "|")
	if len(parts) != 3 {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка обработки выбора."))
		return
	}

	videoURL := parts[1]
	index, err := strconv.Atoi(parts[2])
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка обработки выбора."))
		return
	}

	client := youtube.Client{}
	video, err := client.GetVideoContext(context.Background(), videoURL)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка получения информации о видео"))
		return
	}

	options := getOptions(video)

	if index < 0 || index >= len(options) {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка обработки выбора."))
		return
	}

	selectedOption := options[index]
	bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Вы выбрали качество: %s. Начинаю загрузку...", strconv.Itoa(selectedOption.Quality))))
	DownloadVideoByFormat(bot, callback.Message, video, selectedOption.Format)
}

func DownloadVideoByFormat(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, video *youtube.Video, format *youtube.Format) {
	chatID := msg.Chat.ID
	client := youtube.Client{}

	if format.AudioChannels == 0 {
		var bestAudio *youtube.Format
		for _, f := range video.Formats {
			if strings.HasPrefix(f.MimeType, "audio/") {
				if bestAudio == nil || f.Bitrate > bestAudio.Bitrate {
					bestAudio = &f
				}
			}
		}

		if bestAudio == nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Не удалось найти подходящий аудиоформат"))
			return
		}

		videoFile, err := downloadStream(client, video, format, "video-*.mp4")
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Ошибка скачивания видео"))
			return
		}
		defer os.Remove(videoFile)

		audioFile, err := downloadStream(client, video, bestAudio, "audio-*.mp3")
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Ошибка скачивания аудио"))
			return
		}
		defer os.Remove(audioFile)

		outputFile := fmt.Sprintf("/tmp/%s.mp4", video.ID)
		err = mergeVideoAndAudio(videoFile, audioFile, outputFile)
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Ошибка объединения видео и аудио"))
			return
		}

		sendVideo(bot, chatID, outputFile, video.Title)
		os.Remove(outputFile)
	} else {
		videoFile, err := downloadStream(client, video, format, "video-*.mp4")
		if err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Ошибка скачивания видео"))
			return
		}
		defer os.Remove(videoFile)

		sendVideo(bot, chatID, videoFile, video.Title)
	}
}

func downloadStream(client youtube.Client, video *youtube.Video, format *youtube.Format, pattern string) (string, error) {
	stream, _, err := client.GetStream(video, format)
	if err != nil {
		return "", err
	}
	tmpFile, err := os.CreateTemp("/tmp", pattern)
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err := tmpFile.ReadFrom(stream); err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

func mergeVideoAndAudio(videoFile, audioFile, outputFile string) error {
	cmd := exec.Command("ffmpeg", "-i", videoFile, "-i", audioFile, "-c:v", "copy", "-c:a", "aac", outputFile)
	return cmd.Run()
}

func sendVideo(bot *tgbotapi.BotAPI, chatID int64, filePath, title string) {
	videoMsg := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(filePath))
	videoMsg.Caption = title

	if _, err := bot.Send(videoMsg); err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка отправки видео."))
		log.Printf("Ошибка отправки видео: %v", err)
	}
}
