package woo

import (
	"encoding/json"
)

type Error struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (err *Error) Failure() bool {
	return !err.Success && err.Message != ""
}

func IsError(response []byte) (bool, string) {
	var err Error
	if json.Unmarshal(response, &err) == nil {
		return err.Failure(), err.Message
	}
	return false, ""
}
