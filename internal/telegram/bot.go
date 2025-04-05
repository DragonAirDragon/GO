package telegram

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api    *tgbotapi.BotAPI
	chatID int64
}

func NewBot(token string, chatID int64) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %w", err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	return &Bot{
		api:    bot,
		chatID: chatID,
	}, nil
}

func (b *Bot) SendMessage(text string) error {
	msg := tgbotapi.NewMessage(b.chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	_, err := b.api.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	return nil
}

func (b *Bot) StartCommandListener() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Бот для мониторинга GitHub аккаунта запущен!")
				b.api.Send(msg)
			case "help":
				helpText := "Доступные команды:\n" +
					"/start - Запустить бота\n" +
					"/help - Показать справку\n" +
					"/status - Показать статус мониторинга"
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, helpText)
				b.api.Send(msg)
			case "status":
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Мониторинг активен")
				b.api.Send(msg)
			default:
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Неизвестная команда. Используйте /help для справки.")
				b.api.Send(msg)
			}
		}
	}
}
