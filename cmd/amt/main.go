package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/nenad/amt/internal/telegram"
	"github.com/sethvargo/go-envconfig"
)

type AmtConfig struct {
	Telegram struct {
		Token   string  `env:"TOKEN"`
		ChatIDs []int64 `env:"CHAT_IDS"`
	} `env:",prefix=TELEGRAM_"`
}

func main() {
	_ = godotenv.Overload("../../.env")
	//page := rod.New().MustConnect().MustPage("https://www.wikipedia.org/")
	//page.MustWaitStable().MustScreenshot("a.png")
	cfg := AmtConfig{}
	if err := envconfig.Process(nil, &cfg); err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", cfg)

	client, err := telegram.New(cfg.Telegram.Token)
	if err != nil {
		panic(err)
	}
	if err = client.ReportChatID(); err != nil {
		panic(err)
	}
}
