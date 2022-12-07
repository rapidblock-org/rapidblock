package groupsio

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/chronos-tachyon/rapidblock/internal/httpclient"
)

type List[T any] struct {
	TotalCount    int  `json:"total_count"`
	StartItem     int  `json:"start_item"`
	EndItem       int  `json:"end_item"`
	NextPageToken int  `json:"next_page_token"`
	HasMore       bool `json:"has_more"`
	Data          []T  `json:"data"`
}

type DatabaseRow struct {
	ID        int             `json:"id"`
	TableID   int             `json:"table_id"`
	GroupID   int             `json:"group_id"`
	RowNumber int             `json:"row_num"`
	NumValues int             `json:"num_vals"`
	Created   time.Time       `json:"created"`
	Updated   time.Time       `json:"updated"`
	Values    []DatabaseValue `json:"vals"`
}

type DatabaseValue struct {
	ID        int       `json:"col_id"`
	Type      Type      `json:"col_type"`
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

func (value DatabaseValue) AsString() string {
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
		panic(fmt.Errorf("%#v not implemented", value.Type))
	}
}

func (value DatabaseValue) AsTime() time.Time {
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
		panic(fmt.Errorf("failed to parse %q as time: %w", value.Text, err))

	default:
		panic(fmt.Errorf("%#v not implemented", value.Type))
	}
}

func (value DatabaseValue) AsBool() bool {
	switch value.Type {
	case CheckboxType:
		return value.Checked

	case NumberType:
		return value.Number != 0

	case TextType:
		str := strings.ToLower(value.Text)
		switch str {
		case "":
			return false
		case "0":
			return false
		case "off":
			return false
		case "n":
			return false
		case "no":
			return false
		case "f":
			return false
		case "false":
			return false

		case "1":
			return true
		case "on":
			return true
		case "y":
			return true
		case "yes":
			return true
		case "t":
			return true
		case "true":
			return true

		default:
			panic(fmt.Errorf("failed to parse %q as bool", str))
		}

	default:
		panic(fmt.Errorf("%#v not implemented", value.Type))
	}
}

func (value DatabaseValue) AsSet(choiceNamesByID map[int]string) map[string]struct{} {
	switch value.Type {
	case MultipleChoiceType:
		set := make(map[string]struct{}, len(value.Choices))
		for _, choiceID := range value.Choices {
			choiceName, found := choiceNamesByID[choiceID]
			if !found {
				panic(fmt.Errorf("unknown choice ID %d", choiceID))
			}
			set[choiceName] = struct{}{}
		}
		return set

	default:
		panic(fmt.Errorf("%#v not implemented", value.Type))
	}
}

func ForEach[T any](
	ctx context.Context,
	client *http.Client,
	baseURL *url.URL,
	baseQuery url.Values,
	reqFn func(*http.Request),
	itemfn func(T) error,
) error {
	q := make(url.Values, 1+len(baseQuery))
	for k, v := range baseQuery {
		q[k] = v
	}

	u := *baseURL
	u.RawQuery = q.Encode()
	looping := true
	for looping {
		var list List[T]
		err := httpclient.Do(ctx, client, http.MethodGet, &u, http.NoBody, reqFn, isOK, &list)
		if err != nil {
			return err
		}

		for _, item := range list.Data {
			err = itemfn(item)
			if err != nil {
				return err
			}
		}

		q.Set("page_token", fmt.Sprint(list.NextPageToken))
		u.RawQuery = q.Encode()
		looping = list.HasMore
	}
	return nil
}

func isOK(code int) bool {
	return code == http.StatusOK
}
