package model

type Notify interface {
	PromptForKeys(interactive, verify bool) (ok bool, err error)
	SendMessage(message, title string) error
}
