// Package pknulms implements LMS client for Pukyong National University.
package pknulms

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Notification represents a single notification.
type Notification struct {
	Type           string
	Title          string
	Datetime       string
	Submitted      bool
	Lecture        string
	Professor      string
	PreviewContent string
}

// String returns a string representation of a notification
// in form of {Type: Title}.
func (n *Notification) String() string {
	return fmt.Sprintf("{%s: %s}", n.Type, n.Title)
}

// Client is a wrapper for a single http.Client instance.
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
		return false, err
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

// GetNotificationsByPage returns a slice of notifications for given page.
// The default page size(notifications per page) is 20.
// If you want to change the page size, you can use GetNotifications.
func (c *Client) GetNotificationsByPage(page int) ([]*Notification, error) {
	target := "http://lms.pknu.ac.kr/ilos/mp/mypage_main_list.acl"
	params := url.Values{
		"start":    {strconv.Itoa((page-1)*20 + 1)},
		"display":  {strconv.Itoa(20)},
		"GUBUN":    {""},
		"encoding": {"utf-8"},
	}
	resp, err := c.httpClient.PostForm(target, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return nil, err
	}

	nts := make([]*Notification, 20)

	dateRe := regexp.MustCompile(`^(.+?) \| 마감일\((.+?)\)$`)

	doc.Find(".resultBox li:nth-of-type(2)").Each(func(i int, s *goquery.Selection) {
		var typeText, title, date, lecture, professor, previewContent string
		var submitted bool

		{
			t := strings.SplitN(strings.TrimSpace(s.Find(".site-link").First().Text()), ": ", 2)
			typeText, title = t[0], t[1]
		}

		{
			t := s.Find("span").Map(func(i int, s *goquery.Selection) string {
				return strings.TrimSpace(s.Text())
			})
			previewContent = t[1]
			if typeText == "과제" {
				matches := dateRe.FindStringSubmatch(t[0])
				submitted = matches[1] == "제출"
				date = matches[2]
			} else {
				submitted = false
				date = t[0]
			}
		}

		{
			t := s.Find("div").Last().Find("a").Map(func(i int, s *goquery.Selection) string {
				return strings.TrimSpace(s.Text())
			})
			lecture, professor = t[1], t[0]
		}

		nts[i] = &Notification{
			Type:           typeText,
			Title:          title,
			Datetime:       date,
			Submitted:      submitted,
			Lecture:        lecture,
			Professor:      professor,
			PreviewContent: previewContent,
		}
	})

	return nts, nil
}

// MustGetNotificationsByPage returns a slice of notifications for given page,
// panics when an error has occurred.
func (c *Client) MustGetNotificationsByPage(page int) []*Notification {
	nts, err := c.GetNotificationsByPage(page)
	if err != nil {
		panic(err)
	}
	return nts
}
