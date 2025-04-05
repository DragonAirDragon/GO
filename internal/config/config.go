package config

import (
	"errors"
	"os"
	"strconv"
)

type Config struct {
	TelegramToken       string
	GitHubToken         string
	GitHubUsername      string
	ChatID              int64
	CheckIntervalMinutes int
}

func LoadConfig() (*Config, error) {
	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	if telegramToken == "" {
		return nil, errors.New("TELEGRAM_TOKEN is not set")
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		return nil, errors.New("GITHUB_TOKEN is not set")
	}

	githubUsername := os.Getenv("GITHUB_USERNAME")
	if githubUsername == "" {
		return nil, errors.New("GITHUB_USERNAME is not set")
	}

	chatIDStr := os.Getenv("TELEGRAM_CHAT_ID")
	if chatIDStr == "" {
		return nil, errors.New("TELEGRAM_CHAT_ID is not set")
	}

	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return nil, errors.New("TELEGRAM_CHAT_ID must be a valid integer")
	}

	intervalStr := os.Getenv("CHECK_INTERVAL_MINUTES")
	interval := 15 // Default interval: 15 minutes
	if intervalStr != "" {
		parsedInterval, err := strconv.Atoi(intervalStr)
		if err == nil && parsedInterval > 0 {
			interval = parsedInterval
		}
	}

	return &Config{
		TelegramToken:       telegramToken,
		GitHubToken:         githubToken,
		GitHubUsername:      githubUsername,
		ChatID:              chatID,
		CheckIntervalMinutes: interval,
	}, nil
}
