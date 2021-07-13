package binance

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

const (
	MAX_LEN = 36
	BROKER  = "J6MCRYME"
)

func NewClientOrderID() string {
	out := fmt.Sprintf("x-%s-", BROKER)
	for len(out) < MAX_LEN {
		out += strconv.Itoa(rand.Intn(10))
	}
	return out
}

func NewClientOrderMetadata(metadata string) (string, error) {
	if metadata == "" {
		return NewClientOrderID(), nil
	}
	out := fmt.Sprintf("x-%s-%s-", BROKER, metadata)
	if len(out) > MAX_LEN {
		return "", fmt.Errorf("clientOrderId cannot have more than %d characters", MAX_LEN)
	}
	for len(out) < MAX_LEN {
		out += strconv.Itoa(rand.Intn(10))
	}
	return out, nil
}

// get metadata from order.ClientOrderId
func ParseClientOrderMetadata(order *Order) (string, error) {
	const MIN_LEN = 3
	subs := strings.Split(order.ClientOrderID, "-")
	if len(subs) < MIN_LEN {
		return "", fmt.Errorf("detected %d substrings in clientOrderId, expected at least %d", len(subs), MIN_LEN)
	}
	return subs[2], nil
}
