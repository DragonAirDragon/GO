package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DragonAirDragon/github-tg-bot/internal/config"
	"github.com/DragonAirDragon/github-tg-bot/internal/github"
	"github.com/DragonAirDragon/github-tg-bot/internal/telegram"
	"github.com/DragonAirDragon/github-tg-bot/pkg/utils"
)

func main() {
	utils.LoadEnv()
	
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	githubClient, err := github.NewClient(cfg.GitHubToken)
	if err != nil {
		log.Fatalf("Failed to create GitHub client: %v", err)
	}

	telegramBot, err := telegram.NewBot(cfg.TelegramToken, cfg.ChatID)
	if err != nil {
		log.Fatalf("Failed to create Telegram bot: %v", err)
	}

	if err := telegramBot.SendMessage("GitHub мониторинг запущен! Отслеживаю аккаунт: " + cfg.GitHubUsername); err != nil {
		log.Printf("Failed to send welcome message: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runMonitoring(ctx, cfg, githubClient, telegramBot)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
}

func runMonitoring(ctx context.Context, cfg *config.Config, githubClient *github.Client, telegramBot *telegram.Bot) {
	ticker := time.NewTicker(time.Duration(cfg.CheckIntervalMinutes) * time.Minute)
	defer ticker.Stop()

	repos, err := githubClient.GetRepositories(ctx, cfg.GitHubUsername)
	if err != nil {
		log.Printf("Failed to get initial repositories: %v", err)
		return
	}

	lastRepoCount := len(repos)
	lastCommits := make(map[string]string)

	for _, repo := range repos {
		commits, err := githubClient.GetLatestCommit(ctx, cfg.GitHubUsername, repo.Name)
		if err != nil {
			log.Printf("Failed to get commits for %s: %v", repo.Name, err)
			continue
		}
		if len(commits) > 0 {
			lastCommits[repo.Name] = commits[0].SHA
		}
	}

	log.Printf("Started monitoring GitHub account: %s", cfg.GitHubUsername)
	log.Printf("Initial state: %d repositories", lastRepoCount)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			currentRepos, err := githubClient.GetRepositories(ctx, cfg.GitHubUsername)
			if err != nil {
				log.Printf("Failed to get repositories: %v", err)
				continue
			}

			if len(currentRepos) > lastRepoCount {
				message := "🆕 Обнаружены новые репозитории:\n"

				existingRepoNames := make(map[string]bool)
				for _, repo := range repos {
					existingRepoNames[repo.Name] = true
				}

				for _, repo := range currentRepos {
					if !existingRepoNames[repo.Name] {
						message += "• " + repo.Name + " - " + repo.Description + "\n"
						message += "  URL: " + repo.URL + "\n\n"
					}
				}

				if err := telegramBot.SendMessage(message); err != nil {
					log.Printf("Failed to send new repositories message: %v", err)
				}

				repos = currentRepos
				lastRepoCount = len(currentRepos)
			}

			for _, repo := range currentRepos {
				commits, err := githubClient.GetLatestCommit(ctx, cfg.GitHubUsername, repo.Name)
				if err != nil {
					log.Printf("Failed to get commits for %s: %v", repo.Name, err)
					continue
				}

				if len(commits) > 0 {
					latestCommit := commits[0]
					lastCommitSHA, exists := lastCommits[repo.Name]

					if !exists || lastCommitSHA != latestCommit.SHA {
						message := "📝 Новый коммит в репозитории " + repo.Name + ":\n"
						message += "• Сообщение: " + latestCommit.Message + "\n"
						message += "• Автор: " + latestCommit.Author + "\n"
						message += "• Дата: " + latestCommit.Date + "\n"
						message += "• URL: " + latestCommit.URL + "\n"

						if err := telegramBot.SendMessage(message); err != nil {
							log.Printf("Failed to send new commit message: %v", err)
						}

						lastCommits[repo.Name] = latestCommit.SHA
					}
				}
			}
		}
	}
}
