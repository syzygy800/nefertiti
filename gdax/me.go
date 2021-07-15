//lint:file-ignore ST1006 receiver name should be a reflection of its identity; don't use generic names such as "this" or "self"
package gdax

type Me struct {
	ID string `json:"id"`
}

func (self *Client) GetMe() (*Me, error) {
	var (
		err error
		out Me
	)
	if _, err = self.Client.Request("GET", "/users/self", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
