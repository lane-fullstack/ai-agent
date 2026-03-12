package telegram

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"ai-agent/internal/config"
)

type Bot struct {
	api           *tgbotapi.BotAPI
	defaultChatID int64
}

var BotClient *Bot

func init() {
	cfg := config.Load()
	BotClient = NewBot(cfg)
}

func NewBot(cfg map[string]any) *Bot {

	api, err := tgbotapi.NewBotAPI(config.AsString(cfg["TelegramToken"]))
	if err != nil {
		log.Fatal(err)
	}

	var defaultChatID int64
	chatIDs := config.AsInt64Slice(cfg["ChatIDs"])
	if len(chatIDs) > 0 {
		defaultChatID = chatIDs[0]
	}

	return &Bot{
		api:           api,
		defaultChatID: defaultChatID,
	}
}

func (b *Bot) Send(chatID int64, text string) {

	targetID := chatID
	if targetID == 0 {
		targetID = b.defaultChatID
	}

	if targetID == 0 {
		return
	}

	msg := tgbotapi.NewMessage(targetID, text)
	b.api.Send(msg)
}
