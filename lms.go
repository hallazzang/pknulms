// Package pknulms implements LMS client for Pukyong National University.
package pknulms

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

// Client is a wrapper for a single http.Client instance
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new LMS client.
func NewClient() (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	c := new(Client)
	c.httpClient = &http.Client{
		Transport: tr,
		Jar:       jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return c, nil
}

// MustNewClient attempts to create a new client, panics when an error has occurred.
func MustNewClient() *Client {
	c, err := NewClient()
	if err != nil {
		panic(err)
	}
	return c
}

// Login logs client into LMS.
func (c *Client) Login(id, pw string) (bool, error) {
	target := "https://lms.pknu.ac.kr/ilos/lo/login.acl"
	params := url.Values{
		"returnURL": {""},
		"challenge": {""},
		"response":  {""},
		"usr_id":    {id},
		"usr_pwd":   {pw},
	}

	resp, err := c.httpClient.PostForm(target, params)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, fmt.Errorf("Expected HTTP status code 200, got %d.", resp.StatusCode)
	}

	html, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return !strings.Contains(string(html), "로그인 정보가 일치하지 않습니다."), nil
}

// MustLogin attempts to login, panics when an error has occurred.
func (c *Client) MustLogin(id, pw string) bool {
	result, err := c.Login(id, pw)
	if err != nil {
		panic(err)
	}
	return result
}
