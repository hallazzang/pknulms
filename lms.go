// Package pknulms implements LMS client for Pukyong National University.
package pknulms

import (
	"crypto/tls"
	"errors"
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

// Lecture represents a single lecture.
type Lecture struct {
	Key  string
	Name string
}

// Notification represents a single notification.
// Datetime field might hold different datetime format for different type of notifications.
// Submitted field holds a boolean value whether the assignment has assigned or not
// if the Type of notification is assignment(in string form, "과제") else always false.
type Notification struct {
	Link           string
	Type           string
	Title          string
	Datetime       string
	Submitted      bool
	Lecture        *Lecture
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
	if c, err := NewClient(); err != nil {
		panic(err)
	} else {
		return c
	}
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
		return false, fmt.Errorf("Expected HTTP status code 200, got %d", resp.StatusCode)
	}

	html, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	return !strings.Contains(string(html), "로그인 정보가 일치하지 않습니다."), nil
}

// MustLogin attempts to login, panics when an error has occurred.
func (c *Client) MustLogin(id, pw string) bool {
	if result, err := c.Login(id, pw); err != nil {
		panic(err)
	} else {
		return result
	}
}

// GetNotifications returns a slice of notifications for given start offset and count.
// Note that start offset begins from 1 so the FIRST notification would be at offset 1, not 0.
func (c *Client) GetNotifications(start, count int) (result []*Notification, e error) {
	target := "http://lms.pknu.ac.kr/ilos/mp/mypage_main_list.acl"
	params := url.Values{
		"start":    {strconv.Itoa(start)},
		"display":  {strconv.Itoa(count)},
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

	dateRe := regexp.MustCompile(`^(.+?) \| 마감일\((.+?)\)$`)
	jsStringRe := regexp.MustCompile(`'(.*?)'`)

	doc.Find(".resultBox li:nth-of-type(2)").EachWithBreak(func(i int, s *goquery.Selection) bool {
		var link, typeText, title, date, professor, previewContent string
		var submitted bool
		var lecture Lecture

		a := s.Find(".site-link").First()
		typeText, title = splitString(strings.TrimSpace(a.Text()), ": ")

		href, exists := a.Attr("href")
		if !exists {
			e = fmt.Errorf("Missing 'href' attribute for the site-link tag")
			return false
		}
		link = "http://lms.pknu.ac.kr" + href

		onclick, exists := a.Attr("onclick")
		if !exists {
			e = errors.New("Missing 'onclick' attribute for the site-link tag")
			return false
		}
		matches := jsStringRe.FindAllStringSubmatch(onclick, -1)
		lecture.Key = matches[1][1]

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

		s.Find("div").Last().Find("a").Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if i == 0 {
				professor = text
			} else if i == 1 {
				lecture.Name = text
			}
		})

		result = append(result, &Notification{
			Link:           link,
			Type:           typeText,
			Title:          title,
			Datetime:       date,
			Submitted:      submitted,
			Lecture:        &lecture,
			Professor:      professor,
			PreviewContent: previewContent,
		})

		return true
	})

	return
}

// MustGetNotifications returns a slice of notifications for given start offset and count,
// panics when an error has occurred.
// Note that start offset begins from 1 so the FIRST notification would be at offset 1, not 0.
func (c *Client) MustGetNotifications(start, count int) []*Notification {
	if result, err := c.GetNotifications(start, count); err != nil {
		panic(err)
	} else {
		return result
	}
}

// GetNotificationsByPage returns a slice of notifications for given page.
// The default page size(notifications per page) is 20.
// If you want to change the page size, you can use GetNotifications.
func (c *Client) GetNotificationsByPage(page int) (result []*Notification, e error) {
	return c.GetNotifications((page-1)*20+1, 20)
}

// MustGetNotificationsByPage returns a slice of notifications for given page,
// panics when an error has occurred.
func (c *Client) MustGetNotificationsByPage(page int) []*Notification {
	if result, err := c.GetNotificationsByPage(page); err != nil {
		panic(err)
	} else {
		return result
	}
}
