package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/DragonAirDragon/GO/internal/github"
	"github.com/DragonAirDragon/GO/internal/models"
	"github.com/DragonAirDragon/GO/internal/telegram"
	"github.com/DragonAirDragon/GO/pkg/utils"
)

type MonitoringState struct {
	repos       map[string]bool
	lastCommits map[string]string
	ticker      *time.Ticker
	cancel      context.CancelFunc
}

func main() {
	utils.LoadEnv()
	
	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	if telegramToken == "" {
		log.Fatalf("TELEGRAM_TOKEN is not set")
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		log.Fatalf("GITHUB_TOKEN is not set")
	}

	githubClient, err := github.NewClient(githubToken)
	if err != nil {
		log.Fatalf("Failed to create GitHub client: %v", err)
	}

	telegramBot, err := telegram.NewBot(telegramToken)
	if err != nil {
		log.Fatalf("Failed to create Telegram bot: %v", err)
	}

	go telegramBot.StartCommandListener()

	monitoringStates := make(map[int64]*MonitoringState)
	statesMutex := sync.RWMutex{}

	callbackChan := telegramBot.GetCallbackChannel()
	go func() {
		for callback := range callbackChan {
			log.Printf("Received callback: %s for chat %d, username: %s", callback.Type, callback.ChatID, callback.Username)
			
			statesMutex.Lock()
			
			if callback.Type == "stop" {
				if state, exists := monitoringStates[callback.ChatID]; exists && state.cancel != nil {
					state.cancel()
					if state.ticker != nil {
						state.ticker.Stop()
					}
					delete(monitoringStates, callback.ChatID)
					log.Printf("Monitoring stopped for chat %d", callback.ChatID)
				}
				statesMutex.Unlock()
				continue
			}
			
			if callback.Type == "update" {
				if state, exists := monitoringStates[callback.ChatID]; exists {
					if state.ticker != nil {
						state.ticker.Stop()
					}
					state.ticker = time.NewTicker(time.Duration(callback.Interval) * time.Minute)
					log.Printf("Updated interval to %d minutes for chat %d", callback.Interval, callback.ChatID)
				}
				statesMutex.Unlock()
				continue
			}
			
			if callback.Type == "start" {
				if state, exists := monitoringStates[callback.ChatID]; exists && state.cancel != nil {
					state.cancel()
					if state.ticker != nil {
						state.ticker.Stop()
					}
				}
				
				ctx, cancel := context.WithCancel(context.Background())
				
				state := &MonitoringState{
					repos:       make(map[string]bool),
					lastCommits: make(map[string]string),
					ticker:      time.NewTicker(time.Duration(callback.Interval) * time.Minute),
					cancel:      cancel,
				}
				
				monitoringStates[callback.ChatID] = state
				statesMutex.Unlock()
				
				go runMonitoring(ctx, callback.ChatID, callback.Username, callback.Interval, githubClient, telegramBot, state)
				continue
			}
			
			statesMutex.Unlock()
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	
	statesMutex.Lock()
	for chatID, state := range monitoringStates {
		if state.cancel != nil {
			state.cancel()
		}
		if state.ticker != nil {
			state.ticker.Stop()
		}
		log.Printf("Stopped monitoring for chat %d", chatID)
	}
	statesMutex.Unlock()
}

func runMonitoring(ctx context.Context, chatID int64, username string, interval int, 
	githubClient *github.Client, telegramBot *telegram.Bot, state *MonitoringState) {
	
	repos, err := githubClient.GetRepositories(ctx, username)
	if err != nil {
		log.Printf("Failed to get initial repositories for %s: %v", username, err)
		telegramBot.SendMessage(chatID, "❌ Не удалось получить репозитории для пользователя <b>" + username + "</b>. Проверьте правильность имени пользователя.")
		return
	}

	for _, repo := range repos {
		state.repos[repo.Name] = true
		
		commits, err := githubClient.GetLatestCommit(ctx, username, repo.Name)
		if err != nil {
			log.Printf("Failed to get commits for %s: %v", repo.Name, err)
			continue
		}
		if len(commits) > 0 {
			state.lastCommits[repo.Name] = commits[0].SHA
		}
	}

	log.Printf("Started monitoring GitHub account: %s for chat %d", username, chatID)
	log.Printf("Initial state: %d repositories", len(repos))
	
	telegramBot.SendMessage(chatID, fmt.Sprintf("✅ Мониторинг GitHub аккаунта <b>%s</b> запущен!\n"+
		"Найдено репозиториев: %d\n"+
		"Интервал проверки: %d минут", username, len(repos), interval))

	for {
		select {
		case <-ctx.Done():
			return
		case <-state.ticker.C:
			currentRepos, err := githubClient.GetRepositories(ctx, username)
			if err != nil {
				log.Printf("Failed to get repositories for %s: %v", username, err)
				continue
			}

			var newRepos []string
			for _, repo := range currentRepos {
				if _, exists := state.repos[repo.Name]; !exists {
					newRepos = append(newRepos, repo.Name)
					state.repos[repo.Name] = true
				}
			}

			if len(newRepos) > 0 {
				message := "🆕 Обнаружены новые репозитории:\n"

				for _, repoName := range newRepos {
					var foundRepo *models.Repository
					
					for i := range currentRepos {
						if currentRepos[i].Name == repoName {
							foundRepo = &currentRepos[i]
							break
						}
					}
					
					if foundRepo != nil {
						message += "• " + foundRepo.Name + " - " + foundRepo.Description + "\n"
						message += "  URL: " + foundRepo.URL + "\n\n"
					}
				}

				telegramBot.SendMessage(chatID, message)
			}

			for _, repo := range currentRepos {
				commits, err := githubClient.GetLatestCommit(ctx, username, repo.Name)
				if err != nil {
					log.Printf("Failed to get commits for %s: %v", repo.Name, err)
					continue
				}

				if len(commits) > 0 {
					latestCommit := commits[0]
					lastCommitSHA, exists := state.lastCommits[repo.Name]

					if !exists || lastCommitSHA != latestCommit.SHA {
						message := "📝 Новый коммит в репозитории " + repo.Name + ":\n"
						message += "• Сообщение: " + latestCommit.Message + "\n"
						message += "• Автор: " + latestCommit.Author + "\n"
						message += "• Дата: " + latestCommit.Date + "\n"
						message += "• URL: " + latestCommit.URL + "\n"

						telegramBot.SendMessage(chatID, message)
						state.lastCommits[repo.Name] = latestCommit.SHA
					}
				}
			}
		}
	}
}
