package huobi

import (
	"encoding/json"
)

type Response struct {
	Status string `json:"status"`
}

func IsError(body []byte) (bool, string) {
	var resp Response
	if json.Unmarshal(body, &resp) == nil {
		return resp.Status != "ok", resp.Status
	}
	return false, ""
}
