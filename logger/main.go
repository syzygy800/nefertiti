package logger

import (
	"fmt"
	"log"
	"runtime"
	"strings"

	"github.com/svanas/nefertiti/errors"
	"github.com/svanas/nefertiti/flag"
	"github.com/svanas/nefertiti/model"
	"github.com/svanas/nefertiti/notify"
)

// send an error to StdOut *and* a notification to Pushover/Telegram
func Error(title string, err error, level int64, notifier model.Notify) error {
	pc, file, line, _ := runtime.Caller(1)
	prefix := errors.FormatCaller(pc, file, line)
	msg := fmt.Sprintf("%s %v", prefix, err)

	// exclude common TCP/IP errors that we don't want to notify the user about
	if strings.Contains(err.Error(), "502 Bad Gateway") || strings.Contains(err.Error(), "no such host") || strings.Contains(err.Error(), "network is unreachable") {
		log.Printf("[ERROR] %s", msg)
		return err
	}

	// include stack trace if (a) we have any, and (b) the user has included --debug
	_, ok := err.(*errors.Error)
	if ok && flag.Debug() {
		log.Printf("[ERROR] %s", err.(*errors.Error).ErrorStack(prefix))
	} else {
		log.Printf("[ERROR] %s", msg)
	}

	// send notification to Pushover or Telegram
	if notifier != nil && notify.CanSend(level, notify.ERROR) {
		if e := notifier.SendMessage(msg, fmt.Sprintf("%s - ERROR", title), model.ONCE_PER_MINUTE); e != nil {
			log.Printf("[ERROR] %v", e)
		}
	}

	return err
}

// send a warning to StdOut
func Warn(err error) {
	pc, file, line, _ := runtime.Caller(1)
	log.Printf("[WARNING] %s %v", errors.FormatCaller(pc, file, line), err)
}

// send a warning to StdOut *and* a notification to Pushover/Telegram
func WarnEx(title string, err error, level int64, notifier model.Notify) {
	pc, file, line, _ := runtime.Caller(1)
	log.Printf("[WARNING] %s %v", errors.FormatCaller(pc, file, line), err)
	if notifier != nil && notify.CanSend(level, notify.INFO) {
		if e := notifier.SendMessage(err.Error(), fmt.Sprintf("%s - WARNING", title), model.ALWAYS); e != nil {
			log.Printf("[ERROR] %v", e)
		}
	}
}

func Info(msg string, a ...interface{}) {
	pc, file, line, _ := runtime.Caller(1)
	log.Printf("[INFO] %s %s", errors.FormatCaller(pc, file, line), func() string {
		if a == nil {
			return msg
		} else {
			return fmt.Sprintf(msg, a...)
		}
	}())
}

func InfoEx(title, msg string, level int64, notifier model.Notify) {
	pc, file, line, _ := runtime.Caller(1)
	log.Printf("[INFO] %s %s", errors.FormatCaller(pc, file, line), msg)
	if notifier != nil && notify.CanSend(level, notify.INFO) {
		if e := notifier.SendMessage(msg, fmt.Sprintf("%s - INFO", title), model.ALWAYS); e != nil {
			log.Printf("[ERROR] %v", e)
		}
	}
}
