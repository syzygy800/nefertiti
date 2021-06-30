package model

type Frequency int

const (
	ALWAYS Frequency = iota
	ONCE_PER_MINUTE
)

type Notify interface {
	PromptForKeys(interactive, verify bool) (ok bool, err error)
	SendMessage(message interface{}, title string, frequency Frequency) error
}
