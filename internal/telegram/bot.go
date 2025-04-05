package telegram

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type MonitoringConfig struct {
	GitHubUsername      string
	CheckIntervalMinutes int
	IsActive            bool
}

type Bot struct {
	api              *tgbotapi.BotAPI
	monitoringConfigs map[int64]*MonitoringConfig
	configMutex      sync.RWMutex
	commandHandlers  map[string]func(update tgbotapi.Update)
	updateChan       chan tgbotapi.Update
	callbackChan     chan MonitoringCallback
}

type MonitoringCallback struct {
	Type      string // "start", "stop", "update"
	ChatID    int64
	Username  string
	Interval  int
}

func NewBot(token string) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %w", err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	b := &Bot{
		api:              bot,
		monitoringConfigs: make(map[int64]*MonitoringConfig),
		configMutex:      sync.RWMutex{},
		updateChan:       make(chan tgbotapi.Update, 100),
		callbackChan:     make(chan MonitoringCallback, 100),
	}

	b.commandHandlers = map[string]func(update tgbotapi.Update){
		"start":    b.handleStart,
		"help":     b.handleHelp,
		"status":   b.handleStatus,
		"track":    b.handleTrack,
		"interval": b.handleInterval,
		"stop":     b.handleStop,
	}

	return b, nil
}

func (b *Bot) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	_, err := b.api.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	return nil
}

func (b *Bot) GetCallbackChannel() <-chan MonitoringCallback {
	return b.callbackChan
}

func (b *Bot) StartCommandListener() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID

		if update.Message.IsCommand() {
			command := update.Message.Command()
			if handler, ok := b.commandHandlers[command]; ok {
				handler(update)
			} else {
				b.SendMessage(chatID, "Неизвестная команда. Используйте /help для справки.")
			}
		} else if update.Message.Text != "" {
			username := b.extractGitHubUsername(update.Message.Text)
			if username != "" {
				b.configMutex.Lock()
				if _, exists := b.monitoringConfigs[chatID]; !exists {
					b.monitoringConfigs[chatID] = &MonitoringConfig{
						GitHubUsername:      username,
						CheckIntervalMinutes: 5, // По умолчанию 5 минут
						IsActive:            true,
					}
				} else {
					b.monitoringConfigs[chatID].GitHubUsername = username
					b.monitoringConfigs[chatID].IsActive = true
				}
				b.configMutex.Unlock()

				b.callbackChan <- MonitoringCallback{
					Type:     "start",
					ChatID:   chatID,
					Username: username,
					Interval: b.monitoringConfigs[chatID].CheckIntervalMinutes,
				}

				b.SendMessage(chatID, fmt.Sprintf("Начинаю отслеживать GitHub аккаунт: <b>%s</b>\nИнтервал проверки: %d минут", 
					username, b.monitoringConfigs[chatID].CheckIntervalMinutes))
			}
		}
	}
}

func (b *Bot) extractGitHubUsername(text string) string {
	text = strings.TrimSpace(text)
	
	if !strings.Contains(text, "/") && !strings.Contains(text, " ") {
		return text
	}
	
	urlRegex := regexp.MustCompile(`github\.com/([a-zA-Z0-9_-]+)`)
	matches := urlRegex.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	
	return ""
}

func (b *Bot) handleStart(update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	b.SendMessage(chatID, "Бот для мониторинга GitHub аккаунта запущен!\n\nОтправьте имя пользователя GitHub или URL профиля, чтобы начать отслеживание.")
}

func (b *Bot) handleHelp(update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	helpText := "Доступные команды:\n" +
		"/start - Запустить бота\n" +
		"/help - Показать справку\n" +
		"/track <username> - Начать отслеживание GitHub аккаунта\n" +
		"/interval <минуты> - Установить интервал проверки\n" +
		"/status - Показать статус мониторинга\n" +
		"/stop - Остановить мониторинг\n\n" +
		"Вы также можете просто отправить имя пользователя GitHub или URL профиля."
	b.SendMessage(chatID, helpText)
}

func (b *Bot) handleStatus(update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	
	b.configMutex.RLock()
	config, exists := b.monitoringConfigs[chatID]
	b.configMutex.RUnlock()
	
	if !exists || !config.IsActive {
		b.SendMessage(chatID, "Мониторинг не активен. Используйте /track <username> для начала отслеживания.")
		return
	}
	
	statusText := fmt.Sprintf("Статус мониторинга:\n" +
		"• Отслеживаемый аккаунт: <b>%s</b>\n" +
		"• Интервал проверки: %d минут\n" +
		"• Статус: активен", 
		config.GitHubUsername, config.CheckIntervalMinutes)
	
	b.SendMessage(chatID, statusText)
}

func (b *Bot) handleTrack(update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	args := strings.Fields(update.Message.CommandArguments())
	
	if len(args) < 1 {
		b.SendMessage(chatID, "Пожалуйста, укажите имя пользователя GitHub.\nПример: /track username")
		return
	}
	
	username := args[0]
	
	b.configMutex.Lock()
	if _, exists := b.monitoringConfigs[chatID]; !exists {
		b.monitoringConfigs[chatID] = &MonitoringConfig{
			GitHubUsername:      username,
			CheckIntervalMinutes: 5, // По умолчанию 5 минут
			IsActive:            true,
		}
	} else {
		b.monitoringConfigs[chatID].GitHubUsername = username
		b.monitoringConfigs[chatID].IsActive = true
	}
	interval := b.monitoringConfigs[chatID].CheckIntervalMinutes
	b.configMutex.Unlock()
	
	b.callbackChan <- MonitoringCallback{
		Type:     "start",
		ChatID:   chatID,
		Username: username,
		Interval: interval,
	}
	
	b.SendMessage(chatID, fmt.Sprintf("Начинаю отслеживать GitHub аккаунт: <b>%s</b>\nИнтервал проверки: %d минут", 
		username, interval))
}

func (b *Bot) handleInterval(update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	args := strings.Fields(update.Message.CommandArguments())
	
	if len(args) < 1 {
		b.SendMessage(chatID, "Пожалуйста, укажите интервал проверки в минутах.\nПример: /interval 5")
		return
	}
	
	interval, err := strconv.Atoi(args[0])
	if err != nil || interval < 1 {
		b.SendMessage(chatID, "Пожалуйста, укажите корректное число минут (минимум 1).")
		return
	}
	
	b.configMutex.Lock()
	if _, exists := b.monitoringConfigs[chatID]; !exists {
		b.SendMessage(chatID, "Сначала укажите аккаунт для отслеживания с помощью команды /track <username>")
		b.configMutex.Unlock()
		return
	}
	
	b.monitoringConfigs[chatID].CheckIntervalMinutes = interval
	username := b.monitoringConfigs[chatID].GitHubUsername
	b.configMutex.Unlock()
	
	b.callbackChan <- MonitoringCallback{
		Type:     "update",
		ChatID:   chatID,
		Username: username,
		Interval: interval,
	}
	
	b.SendMessage(chatID, fmt.Sprintf("Интервал проверки для аккаунта <b>%s</b> установлен на %d минут", 
		username, interval))
}

func (b *Bot) handleStop(update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	
	b.configMutex.Lock()
	config, exists := b.monitoringConfigs[chatID]
	if !exists || !config.IsActive {
		b.SendMessage(chatID, "Мониторинг уже остановлен.")
		b.configMutex.Unlock()
		return
	}
	
	config.IsActive = false
	username := config.GitHubUsername
	b.configMutex.Unlock()
	
	b.callbackChan <- MonitoringCallback{
		Type:     "stop",
		ChatID:   chatID,
		Username: username,
	}
	
	b.SendMessage(chatID, fmt.Sprintf("Мониторинг аккаунта <b>%s</b> остановлен.", username))
}
