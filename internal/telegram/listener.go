package telegram

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"ai-agent/internal/scheduler"
)

func StartListener(bot *Bot, s *scheduler.Scheduler) {

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.api.GetUpdatesChan(u)

	for update := range updates {

		if update.Message == nil {
			continue
		}

		text := update.Message.Text

		if strings.HasPrefix(text, "/ping") {
			bot.Send(update.Message.Chat.ID, "task manager online")
		}
	}
}
