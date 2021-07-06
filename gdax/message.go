package gdax

import (
	"strings"

	exchange "github.com/svanas/go-coinbasepro"
)

//-------------------------------- MessageType --------------------------------

type MessageType int

const (
	MESSAGE_UNKNOWN MessageType = iota
	MESSAGE_RECEIVED
	MESSAGE_OPEN
	MESSAGE_DONE
	MESSAGE_MATCH
	MESSAGE_CHANGE
	MESSAGE_ACTIVATE
	MESSAGE_HEARTBEAT
	MESSAGE_ERROR
)

var messageTypeString = map[MessageType]string{
	MESSAGE_UNKNOWN:   "",
	MESSAGE_RECEIVED:  "received",
	MESSAGE_OPEN:      "open",
	MESSAGE_DONE:      "done",
	MESSAGE_MATCH:     "match",
	MESSAGE_CHANGE:    "change",
	MESSAGE_ACTIVATE:  "activate",
	MESSAGE_HEARTBEAT: "heartbeat",
	MESSAGE_ERROR:     "error",
}

func (mt *MessageType) String() string {
	return messageTypeString[*mt]
}

//------------------------------- MessageReason -------------------------------

type MessageReason int

const (
	REASON_UNKNOWN MessageReason = iota
	REASON_FILLED
	REASON_CANCELED
)

var messageReasonString = map[MessageReason]string{
	REASON_UNKNOWN:  "",
	REASON_FILLED:   "filled",
	REASON_CANCELED: "canceled",
}

func (mr *MessageReason) String() string {
	return messageReasonString[*mr]
}

//---------------------------------- Message ----------------------------------

type Message struct {
	*exchange.Message
}

func (self *Message) Title() string {
	out := "Coinbase Pro - " + strings.Title(self.Type) + " " + strings.Title(self.Side)
	if self.GetType() == MESSAGE_DONE {
		if self.Reason != "" {
			out = out + " (Reason: " + strings.Title(self.Reason) + ")"
		}
	}
	return out
}

func (self *Message) GetType() MessageType {
	for mt := range messageTypeString {
		if mt.String() == self.Type {
			return mt
		}
	}
	return MESSAGE_UNKNOWN
}

func (self *Message) GetReason() MessageReason {
	for mr := range messageReasonString {
		if mr.String() == self.Reason {
			return mr
		}
	}
	return REASON_UNKNOWN
}
