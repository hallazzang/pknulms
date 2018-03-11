package pknulms

import (
	"encoding/json"
	"errors"
	"net/url"
)

// SendNote sends note to a person with given title and content.
func (c *Client) SendNote(to, title, content string) error {
	target := "http://lms.pknu.ac.kr/ilos/message/insert_pop.acl"
	params := url.Values{
		"TITLE":    {title},
		"RECV_IDs": {to + "^"},
		"CONTENT":  {content},
		"encoding": {"utf-8"},
	}
	resp, err := c.httpClient.PostForm(target, params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		IsError bool   `json:"isError"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	// Actually, it seems that an error cannot occur here
	if result.IsError {
		return errors.New(result.Message)
	}

	return nil
}

// MustSendNote sends note to a person with given title and content,
// panics when an error has occurred.
func (c *Client) MustSendNote(to, title, content string) {
	if err := c.SendNote(to, title, content); err != nil {
		panic(err)
	}
}
