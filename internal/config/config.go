package config

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	TelegramToken string  `json:"TelegramToken"`
	ChatIDs       []int64 `json:"ChatIDs"`
	DBPath        string  `json:"DBPath"`
	TranslateAPI  string  `json:"TranslateAPI"`
	TranslateKey  string  `json:"TranslateKey"`
	ResumePath    string  `json:"ResumePath"`
	GeminiAPIKey  string  `json:"GeminiAPIKey"`
}

func Load() Config {
	file, err := os.Open("config.json")
	if err != nil {
		log.Fatal("Error opening config.json: ", err)
	}
	defer file.Close()

	var cfg Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&cfg)
	if err != nil {
		log.Fatal("Error decoding config.json: ", err)
	}

	if cfg.TranslateAPI == "" {
		cfg.TranslateAPI = "http://localhost:5050/translate"
	}

	return cfg
}
