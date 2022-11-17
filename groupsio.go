package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type GIOList[T any] struct {
	TotalCount    int  `json:"total_count"`
	StartItem     int  `json:"start_item"`
	EndItem       int  `json:"end_item"`
	NextPageToken int  `json:"next_page_token"`
	HasMore       bool `json:"has_more"`
	Data          []T  `json:"data"`
}

type GIODatabaseRow struct {
	ID        int                `json:"id"`
	TableID   int                `json:"table_id"`
	GroupID   int                `json:"group_id"`
	RowNumber int                `json:"row_num"`
	NumValues int                `json:"num_vals"`
	Created   time.Time          `json:"created"`
	Updated   time.Time          `json:"updated"`
	Values    []GIODatabaseValue `json:"vals"`
}

type GIODatabaseValue struct {
	ID        int       `json:"col_id"`
	Type      GIOType   `json:"col_type"`
	Choices   []int     `json:"multi_choice"`
	Text      string    `json:"text"`
	HTMLText  string    `json:"html_text"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	ImageName string    `json:"image_name"`
	Date      time.Time `json:"date"`
	Time      time.Time `json:"time"`
	Number    int64     `json:"number"`
	Checked   bool      `json:"checked"`
}

func (value GIODatabaseValue) AsString() string {
	switch value.Type {
	case TextType:
		return value.Text
	case ParagraphType:
		return value.Text
	case HTMLParagraphType:
		return value.HTMLText
	case CheckboxType:
		return strconv.FormatBool(value.Checked)
	case NumberType:
		return strconv.FormatInt(value.Number, 10)
	case DateType:
		return value.Date.Format("2006-01-02")
	case TimeType:
		return value.Time.Format(time.RFC3339Nano)
	case LinkType:
		return value.URL

	default:
		fmt.Fprintf(os.Stderr, "fatal: %#v is not implemented\n", value.Type)
		os.Exit(1)
		return ""
	}
}

func (value GIODatabaseValue) AsTime() time.Time {
	switch value.Type {
	case DateType:
		return value.Date

	case TimeType:
		return value.Time

	case TextType:
		var t time.Time
		var err error
		for _, layout := range []string{"2006-01-02", time.RFC3339, time.RFC3339Nano} {
			t, err = time.ParseInLocation(layout, value.Text, time.UTC)
			if err == nil {
				return t
			}
		}
		fmt.Fprintf(os.Stderr, "fatal: failed to parse %q as time: %v\n", value.Text, err)
		os.Exit(1)
		return time.Time{}

	default:
		fmt.Fprintf(os.Stderr, "fatal: %#v is not implemented\n", value.Type)
		os.Exit(1)
		return time.Time{}
	}
}

func (value GIODatabaseValue) AsBool() bool {
	switch value.Type {
	case CheckboxType:
		return value.Checked

	case NumberType:
		return value.Number != 0

	case TextType:
		str := strings.ToLower(value.Text)
		switch str {
		case "":
			fallthrough
		case "0":
			fallthrough
		case "n":
			fallthrough
		case "no":
			fallthrough
		case "f":
			fallthrough
		case "false":
			return false

		case "1":
			fallthrough
		case "y":
			fallthrough
		case "yes":
			fallthrough
		case "t":
			fallthrough
		case "true":
			return true
		}
		fmt.Fprintf(os.Stderr, "fatal: failed to parse %q as bool\n", str)
		os.Exit(1)
		return false

	default:
		fmt.Fprintf(os.Stderr, "fatal: %#v is not implemented\n", value.Type)
		os.Exit(1)
		return false
	}
}

func (value GIODatabaseValue) AsSet(choiceNamesByID map[int]string) map[string]struct{} {
	switch value.Type {
	case MultipleChoiceType:
		set := make(map[string]struct{}, len(value.Choices))
		for _, choiceID := range value.Choices {
			choiceName, found := choiceNamesByID[choiceID]
			if !found {
				fmt.Fprintf(os.Stderr, "fatal: unknown choice ID %d\n", choiceID)
				os.Exit(1)
			}
			set[choiceName] = struct{}{}
		}
		return set

	default:
		fmt.Fprintf(os.Stderr, "fatal: %#v is not implemented\n", value.Type)
		os.Exit(1)
		return nil
	}
}

func GIOForEach[T any](
	ctx context.Context,
	client *http.Client,
	baseURL *url.URL,
	baseQuery url.Values,
	reqfn func(*http.Request),
	itemfn func(T) error,
) error {
	u := *baseURL
	q := make(url.Values, 1+len(baseQuery))
	for k, v := range baseQuery {
		q[k] = v
	}
	u.RawQuery = q.Encode()
	urlstr := u.String()

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlstr, http.NoBody)
		if err != nil {
			return fmt.Errorf("%s: %s: failed to create request: %w", http.MethodGet, urlstr, err)
		}

		reqfn(req)

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("%s: %s: request failed: %w", http.MethodGet, urlstr, err)
		}

		rawBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			_ = resp.Body.Close()
			return fmt.Errorf("%s: %s: I/O error in response body: %w", http.MethodGet, urlstr, err)
		}

		err = resp.Body.Close()
		if err != nil {
			return fmt.Errorf("%s: %s: I/O error in response body: %w", http.MethodGet, urlstr, err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("%s: %s: unexpected status %03d", http.MethodGet, urlstr, resp.StatusCode)
		}

		var list GIOList[T]
		err = json.Unmarshal(rawBody, &list)
		if err != nil {
			return fmt.Errorf("%s: %s: failed to decode response body as JSON: %w", http.MethodGet, urlstr, err)
		}

		for _, item := range list.Data {
			err = itemfn(item)
			if err != nil {
				return fmt.Errorf("%s: %s: %w", http.MethodGet, urlstr, err)
			}
		}

		if !list.HasMore {
			return nil
		}

		q.Set("page_token", fmt.Sprint(list.NextPageToken))
		u.RawQuery = q.Encode()
		urlstr = u.String()
	}
}
