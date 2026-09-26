package main

import (
	"fmt"
	"log"
	"os"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/memory"
	"github.com/mackerelio/go-osstat/uptime"
	"github.com/showwin/speedtest-go/speedtest"
)

var numericKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Status"),
		tgbotapi.NewKeyboardButton("SpeedTest"),
	),
)

func getSpeedTest() string {
	client := speedtest.New()

	serverList, err := client.FetchServers()
	if err != nil {
		return "Не удалось получить список серверов speedtest"
	}

	targets, err := serverList.FindServer(nil)
	if err != nil || len(targets) == 0 {
		return "Не удалось найти сервер для теста"
	}

	targetServer := targets[0]

	err = targetServer.PingTest(nil)
	if err != nil {
		return "Ошибка пинга"
	}

	err = targetServer.DownloadTest()
	if err != nil {
		return "Ошибка теста Download"
	}

	err = targetServer.UploadTest()
	if err != nil {
		return "Ошибка теста Upload"
	}

	dlMB := targetServer.DLSpeed / 8
	ulMB := targetServer.ULSpeed / 8

	return fmt.Sprintf(" <b>SpeedTest (%s):</b>\n\n Download: <b>%.2f MB/s</b> (%.2f Mbps)\n Upload: <b>%.2f MB/s</b> (%.2f Mbps)\n Ping: <b>%v</b>",
		targetServer.Name,
		dlMB, targetServer.DLSpeed,
		ulMB, targetServer.ULSpeed,
		targetServer.Latency,
	)
}

func getDiskStats(path string) string {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return "Ошибка чтения диска"
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := total - free

	usedGB := used / 1024 / 1024 / 1024
	totalGB := total / 1024 / 1024 / 1024

	return fmt.Sprintf("Disk: %d / %d GB", usedGB, totalGB)
}

func getServerStats() string {
	memStr := "Ошибка чтения памяти"
	if memStat, err := memory.Get(); err == nil {
		usedMB := memStat.Used / 1024 / 1024
		totalMB := memStat.Total / 1024 / 1024
		memStr = fmt.Sprintf("RAM: %d / %d MB", usedMB, totalMB)
	}

	cpuStr := "Ошибка чтения CPU"
	cpuStat1, _ := cpu.Get()
	time.Sleep(500 * time.Millisecond)
	cpuStat2, _ := cpu.Get()

	totalDelta := float64(cpuStat2.Total - cpuStat1.Total)
	idleDelta := float64(cpuStat2.Idle - cpuStat1.Idle)
	if totalDelta > 0 {
		usage := 100.0 * (1.0 - idleDelta/totalDelta)
		cpuStr = fmt.Sprintf("CPU: %.1f%%", usage)
	}

	upStr := "Ошибка чтения аптайма"
	if upStat, err := uptime.Get(); err == nil {
		upStr = fmt.Sprintf("Uptime: %v", upStat)
	}

	diskStr := getDiskStats("/hostfs")

	loc, err := time.LoadLocation("Europe/Moscow")
	var currentTime string
	if err != nil {
		currentTime = time.Now().Format("02.01.2006 15:04:05")
	} else {
		currentTime = time.Now().In(loc).Format("02.01.2006 15:04:05")
	}

	return fmt.Sprintf("<b>Статус сервера:</b>\n\n %s\n %s\n %s\n %s\n\n <i>Время: %s</i>",
		cpuStr, memStr, diskStr, upStr, currentTime)
}

func main() {
	apiKey := os.Getenv("TG_BOT_TOKEN")
	if apiKey == "" {
		log.Fatal("Переменная TG_BOT_TOKEN не задана")
	}

	bot, err := tgbotapi.NewBotAPI(apiKey)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
		msg.ParseMode = "HTML"

		switch update.Message.Text {
		case "open", "/start":
			msg.Text = "Привет! Выбери действие на клавиатуре 👇"
			msg.ReplyMarkup = numericKeyboard
			bot.Send(msg)

		case "Status":
			msg.Text = getServerStats()
			bot.Send(msg)

		case "SpeedTest":

			msg.Text = "⏳ <i>Замеряю скорость интернета на сервере, подожди 5-10 секунд...</i>"
			sentMsg, err := bot.Send(msg)
			if err != nil {
				continue
			}

			go func(chatId int64, messageId int) {
				resultText := getSpeedTest()

				editMsg := tgbotapi.NewEditMessageText(chatId, messageId, resultText)
				editMsg.ParseMode = "HTML"
				bot.Send(editMsg)
			}(update.Message.Chat.ID, sentMsg.MessageID)

		default:
			msg.Text = "Неизвестная команда. Используй клавиатуру."
			bot.Send(msg)
		}
	}
}
