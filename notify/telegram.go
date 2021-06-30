package notify

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/svanas/nefertiti/flag"
	"github.com/svanas/nefertiti/model"
	"github.com/svanas/nefertiti/passphrase"
	"github.com/yanzay/tbot"
)

type Telegram struct {
	appKey string
	chatId int64
}

func (this *Telegram) PromptForKeys(interactive, verify bool) (ok bool, err error) {
	this.appKey = flag.Get("telegram-app-key").String()

	if this.appKey == "none" {
		return false, nil
	}

	if this.appKey == "" && interactive {
		var data []byte
		if data, err = passphrase.Read("Telegram application key"); err != nil {
			return false, err
		}
		this.appKey = string(data)
	}

	if this.appKey == "none" {
		return false, nil
	}

	if this.appKey != "" {
		chatId := flag.Get("telegram-chat-id").String()
		if chatId != "" {
			this.chatId, err = strconv.ParseInt(chatId, 10, 64)
		} else {
			if interactive {
				var data []byte
				if data, err = passphrase.Read("Telegram chat ID"); err != nil {
					return false, err
				}
				this.chatId, err = strconv.ParseInt(string(data), 10, 64)
			}
		}
		if err != nil {
			return false, errors.New("error when getting Telegram chat ID")
		}
	}

	return this.appKey != "" && this.chatId != 0, nil
}

func (this *Telegram) SendMessage(message interface{}, title string, frequency model.Frequency) error {
	if this.appKey == "" {
		return errors.New("missing argument: Telegram application key")
	}
	if this.chatId == 0 {
		return errors.New("missing argument: Telegram chatId")
	}

	var (
		err  error
		body string
	)
	if body, err = func(message interface{}) (string, error) {
		if str, ok := message.(string); ok {
			return str, nil
		} else {
			data, err := json.Marshal(message)
			if err != nil {
				return "", err
			} else {
				return string(data), nil
			}
		}
	}(message); err != nil {
		return err
	}

	bot, err := tbot.NewServer(this.appKey)
	if err != nil {
		return err
	}

	if title == "" {
		return bot.Send(this.chatId, body)
	} else {
		return bot.Send(this.chatId, (title + ": " + body))
	}
}

func NewTelegram() model.Notify {
	return &Telegram{}
}
