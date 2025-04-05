package utils

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		dir, err := os.Getwd()
		if err != nil {
			log.Printf("Warning: Failed to get current directory: %v", err)
			return
		}

		for i := 0; i < 3; i++ { // Проверяем до 3 уровней вверх
			dir = filepath.Dir(dir)
			envPath := filepath.Join(dir, ".env")
			if _, err := os.Stat(envPath); err == nil {
				err = godotenv.Load(envPath)
				if err == nil {
					log.Printf("Loaded .env from %s", envPath)
					return
				}
			}
		}

		log.Printf("Warning: .env file not found, using environment variables")
	} else {
		log.Println("Loaded .env file from current directory")
	}
}
