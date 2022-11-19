package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"time"
)

const UserAgentFormat = "RapidBlock/%s (+https://github.com/chronos-tachyon/rapidblock/)"

type AccountData struct {
	Cookies map[string]string  `json:"cookies"`
	Columns map[int]ColumnData `json:"columns"`
}

type ColumnData struct {
	ID      ColumnID       `json:"id"`
	Choices map[int]string `json:"choices"`
}

func (ad AccountData) CookieString() (string, bool) {
	if len(ad.Cookies) <= 0 {
		return "", false
	}

	var buf bytes.Buffer
	isNext := false
	for name, value := range ad.Cookies {
		if isNext {
			buf.WriteString("; ")
		}
		buf.WriteString(name)
		buf.WriteByte('=')
		buf.WriteString(value)
		isNext = true
	}
	return buf.String(), true
}

func cmdPrepareData() {
	switch {
	case flagAccountDataFile == "":
		fmt.Fprintf(os.Stderr, "fatal: missing required flag -A / --account-data-file\n")
		os.Exit(1)
	case flagSourceID == "":
		fmt.Fprintf(os.Stderr, "fatal: missing required flag -S / --source-id\n")
		os.Exit(1)
	case flagDataFile == "":
		fmt.Fprintf(os.Stderr, "fatal: missing required flag -d / --data-file\n")
		os.Exit(1)
	}

	var ad AccountData
	ReadJsonFile(&ad, flagAccountDataFile)

	var file BlockFile
	file.Spec = BlockFileSpecV1
	file.PublishedAt = time.Now().UTC()
	file.Blocks = make(map[string]Block, 1024)

	baseURL := &url.URL{
		Scheme:  "https",
		Host:    "groups.io",
		Path:    "/api/v1/getdatabaserows",
		RawPath: "/api/v1/getdatabaserows",
	}
	baseQuery := make(url.Values, 2)
	baseQuery.Set("database_id", flagSourceID)
	baseQuery.Set("limit", "100")

	userAgent := fmt.Sprintf(UserAgentFormat, Version)
	cookie, hasCookie := ad.CookieString()

	ctx := context.Background()
	err := GIOForEach(
		ctx,
		http.DefaultClient,
		baseURL,
		baseQuery,
		func(req *http.Request) {
			if req.Header == nil {
				req.Header = make(http.Header, 16)
			}
			req.Header.Set("user-agent", userAgent)
			if hasCookie {
				req.Header.Set("cookie", cookie)
			}
		},
		func(row GIODatabaseRow) error {
			var block Block
			var domain string
			var hasDomain bool

			for _, value := range row.Values {
				columnData := ad.Columns[value.ID]
				switch columnData.ID {
				case DomainID:
					domain = value.AsString()
					hasDomain = true
				case IsBlockedID:
					block.IsBlocked = value.AsBool()
				case DateRequestedID:
					block.DateRequested = value.AsTime()
				case DateDecidedID:
					block.DateDecided = value.AsTime()
				case ReasonID:
					block.Reason = value.AsString()
				case TagsID:
					block.Tags = sortTags(value.AsSet(columnData.Choices))
				}
			}

			if hasDomain && !block.DateDecided.IsZero() && !block.DateDecided.After(file.PublishedAt) {
				file.Blocks[domain] = block
			}
			return nil
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}

	WriteJsonFile(flagDataFile, file, false)
}

func sortTags(set map[string]struct{}) []string {
	list := make([]string, 0, len(set))
	for tag := range set {
		list = append(list, tag)
	}
	sort.Strings(list)
	return list
}
