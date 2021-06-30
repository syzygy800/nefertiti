package notify

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/gregdel/pushover"
	"github.com/svanas/nefertiti/flag"
	"github.com/svanas/nefertiti/model"
	"github.com/svanas/nefertiti/passphrase"
)

type Pushover struct {
	appKey  string
	userKey string
}

func (this *Pushover) promptForKeys(interactive bool) (ok bool, err error) {
	this.appKey = flag.Get("pushover-app-key").String()

	if this.appKey == "none" {
		return false, nil
	}

	if this.appKey == "" && interactive {
		var data []byte
		if data, err = passphrase.Read("Pushover application key"); err != nil {
			return false, err
		}
		this.appKey = string(data)
	}

	if this.appKey == "none" {
		return false, nil
	}

	if this.appKey != "" {
		this.userKey = flag.Get("pushover-user-key").String()
		if this.userKey == "" && interactive {
			var data []byte
			if data, err = passphrase.Read("Pushover user key"); err != nil {
				return false, err
			}
			this.userKey = string(data)
		}
	}

	return this.appKey != "" && this.userKey != "", nil
}

func (this *Pushover) PromptForKeys(interactive, verify bool) (ok bool, err error) {
	ok, err = this.promptForKeys(interactive)
	if ok && verify {
		// verify the pushover user key
		if err = pushoverVerifyRecipient(this.appKey, this.userKey); err != nil {
			ok = false
		}
	}
	return
}

func pushoverVerifyRecipient(appToken, userToken string) error {
	if appToken == "" {
		return errors.New("missing argument: Pushover application key")
	}
	app := pushover.New(appToken)
	rec := pushover.NewRecipient(userToken)
	_, err := app.GetRecipientDetails(rec)
	return err
}

var (
	messageHistory map[string]time.Time
)

func init() {
	messageHistory = make(map[string]time.Time)
}

func (this *Pushover) SendMessage(message interface{}, title string, frequency model.Frequency) error {
	if this.appKey == "" {
		return errors.New("missing argument: Pushover application key")
	}
	if this.userKey == "" {
		return errors.New("missing argument: Pushover recipient")
	}

	var (
		err       error
		body      string
		monospace bool
	)
	if body, monospace, err = func(message interface{}) (string, bool, error) { // -> (string, monospace, error)
		if str, ok := message.(string); ok {
			return str, false, nil
		} else {
			data, err := json.MarshalIndent(message, "", "  ")
			if err != nil {
				return "", false, err
			} else {
				return string(data), true, nil
			}
		}
	}(message); err != nil {
		return err
	}

	sent, ok := messageHistory[body]
	if ok {
		if frequency != model.ALWAYS {
			elapsed := time.Since(sent)
			if frequency == model.ONCE_PER_MINUTE && elapsed.Minutes() < 1 {
				return nil
			}
		}
		delete(messageHistory, body)
	}

	app := pushover.New(this.appKey)
	rec := pushover.NewRecipient(this.userKey)

	msg := pushover.NewMessageWithTitle(body, title)
	msg.Monospace = monospace

	_, err = app.SendMessage(msg, rec)
	if err != nil {
		return err
	} else {
		messageHistory[body] = time.Now()
	}

	return nil
}

func NewPushover() model.Notify {
	return &Pushover{}
}
