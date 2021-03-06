package pknulms

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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
	ID             int
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

// GetNotifications returns a slice of notifications for given start offset and count.
// Note that start offset begins from 1 so the FIRST notification would be at offset 1, not 0.
// Weirdly, it seems that the count must be >= 8 because of some mysterious reasons.
func (c *Client) GetNotifications(start, count int) (result []*Notification, e error) {
	if count < 8 {
		return nil, errors.New("Count must be >= 8")
	}

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
	idRe := regexp.MustCompile(`=(\d+)$`)
	jsStringRe := regexp.MustCompile(`'(.*?)'`)

	doc.Find(".resultBox li:nth-of-type(2)").EachWithBreak(func(i int, s *goquery.Selection) bool {
		var id int
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
		idMatches := idRe.FindStringSubmatch(href)
		if len(idMatches) == 0 {
			e = fmt.Errorf("Mismatching 'href' pattern: %s", href)
			return false
		}
		id, err := strconv.Atoi(idMatches[1])
		if err != nil {
			e = err
			return false
		}
		link = "http://lms.pknu.ac.kr" + href

		onclick, exists := a.Attr("onclick")
		if !exists {
			e = errors.New("Missing 'onclick' attribute for the site-link tag")
			return false
		}
		jsStringMatches := jsStringRe.FindAllStringSubmatch(onclick, -1)
		if len(jsStringMatches) < 2 || len(jsStringMatches[1]) < 2 {
			e = fmt.Errorf("Mismatching 'onclick' pattern: %s", onclick)
			return false
		}
		lecture.Key = jsStringMatches[1][1]

		t := s.Find("span").Map(func(i int, s *goquery.Selection) string {
			return strings.TrimSpace(s.Text())
		})
		previewContent = t[1]
		if typeText == "과제" {
			dateMatches := dateRe.FindStringSubmatch(t[0])
			if len(dateMatches) < 3 {
				e = fmt.Errorf("Mismatching dateText pattern: %s", t[0])
				return false
			}
			submitted = dateMatches[1] == "제출"
			date = dateMatches[2]
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
			ID:             id,
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

// prefetchArticle requests to prefetch an article.
func (c *Client) prefetchArticle(key, returnURL string) error {
	target := "http://lms.pknu.ac.kr/ilos/st/course/eclass_room2.acl"
	params := url.Values{
		"KJKEY":     {key},
		"returnURI": {returnURL},
		"encoding":  {"utf-8"},
	}
	resp, err := c.httpClient.PostForm(target, params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	type Result struct {
		IsError     bool   `json:"isError"`
		Message     string `json:"message"`
		LectureType string `json:"lectType"`
		ReturnURL   string `json:"returnURL"`
	}
	var result Result

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, &result); err != nil {
		panic(err)
	}
	if result.IsError {
		return errors.New(result.Message)
	}

	return nil
}

// GetNotificationContent returns content of given notification.
// The result contains HTML codes, not plain text.
func (c *Client) GetNotificationContent(n *Notification) (string, error) {
	err := c.prefetchArticle(n.Lecture.Key,
		strings.TrimPrefix(n.Link, "http://lms.pknu.ac.kr"))
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Get(n.Link + "&s=menu&acl=")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return "", err
	}

	lines := doc.Find(".bbsview .textviewer").Contents().Map(func(i int, s *goquery.Selection) string {
		switch goquery.NodeName(s) {
		case "script":
			return ""
		case "#text":
			return strings.TrimSpace(s.Text())
		default:
			html, _ := goquery.OuterHtml(s)
			return strings.TrimSpace(html)
		}
	})

	return strings.Join(filterNotEmptyString(lines), "\n"), nil
}

// MustGetNotificationContent returns content of given notification.
// The result contains HTML codes, not plain text.
func (c *Client) MustGetNotificationContent(n *Notification) string {
	result, err := c.GetNotificationContent(n)
	if err != nil {
		panic(err)
	}
	return result
}
