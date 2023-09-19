package telegram

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"strconv"
)

type Client struct {
	token string
	chats []int64

	bot *tgbotapi.BotAPI
}

// New creates a telegram client
func New(token string, chats ...int64) (*Client, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &Client{token: token, chats: chats, bot: bot}, nil
}

// Send a message to the chats
func (c *Client) Send(text string) error {
	for _, chat := range c.chats {
		msg := tgbotapi.NewMessage(chat, text)
		_, err := c.bot.Send(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) ReportChatID() error {
	fmt.Println("Listening for new messages...")
	fmt.Printf("Text the @%s bot to get your chat ID\n", c.bot.Self.UserName)
	fmt.Println("Press Ctrl+C to stop")
	allUpdates, err := c.bot.GetUpdates(tgbotapi.NewUpdate(0))
	if err != nil {
		return fmt.Errorf("could not get all messages: %s", err)
	}
	lastOffset := 0
	for _, upd := range allUpdates {
		if upd.UpdateID > lastOffset {
			lastOffset = upd.UpdateID
		}
	}

	// We are listening only for new messages
	updChan := c.bot.GetUpdatesChan(tgbotapi.NewUpdate(lastOffset + 1))
	defer c.bot.StopReceivingUpdates()
	for upd := range updChan {
		if upd.Message != nil {
			msg := "Your chat ID is: " + strconv.FormatInt(upd.Message.Chat.ID, 10)
			fmt.Println(msg)
			tgMsg := tgbotapi.NewMessage(upd.Message.Chat.ID, msg)
			if _, err = c.bot.Send(tgMsg); err != nil {
				return fmt.Errorf("could not send telegram chat ID: %s", err)
			}
		}
	}

	return nil
}
