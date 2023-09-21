package main

import (
	"flag"
	"github.com/joho/godotenv"
	"github.com/nenad/amt/internal/amts/lea"
	"github.com/nenad/amt/internal/config"
	"github.com/nenad/amt/internal/telegram"
	"github.com/sethvargo/go-envconfig"
	"strings"
)

var (
	validServices = []string{"lea"}
	envFlag       = flag.String("env", ".env", "Path to .env file")
	serviceFlag   = flag.String("service", "lea", "Service to use. Valid: "+strings.Join(validServices, ", "))
)

func main() {
	// Load .env file
	flag.Parse()
	if err := godotenv.Overload(*envFlag); err != nil {
		panic("Error loading .env file - maybe you need to create one? See README.md for more information.")
	}
	cfg := config.AmtConfig{}
	if err := envconfig.Process(nil, &cfg); err != nil {
		panic(err)
	}

	telegramClient, err := telegram.New(cfg.Telegram.Token, cfg.Telegram.ChatIDs...)
	if err != nil {
		panic("Could not create Telegram client: " + err.Error())
	}

	switch *serviceFlag {
	case "lea":
		leaScenario := lea.LeaScenario{
			TelegramClient: telegramClient,
		}
		if err = leaScenario.Run(cfg.Lea); err != nil {
			panic("Error running LEA scenario: " + err.Error())
		}
	default:
		panic("Invalid service; valid: " + strings.Join(validServices, ","))
	}

}
