package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

func loadServiceAccount(filePath string, scopes ...string) *jwt.Config {
	raw := ReadFile(filePath)
	config, err := google.JWTConfigFromJSON(raw, scopes...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %q: failed to load service account credentials: %v\n", filePath, err)
		os.Exit(1)
	}
	return config
}

var dialer = net.Dialer{
	Timeout:       15 * time.Second,
	KeepAlive:     15 * time.Second,
	FallbackDelay: 300 * time.Millisecond,
}

var transport = http.Transport{
	Proxy:                 http.ProxyFromEnvironment,
	DialContext:           dialer.DialContext,
	MaxIdleConns:          100,
	MaxIdleConnsPerHost:   100,
	MaxConnsPerHost:       100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 10 * time.Second,
}

func init() {
	h2, err := http2.ConfigureTransports(&transport)
	if err != nil {
		panic(err)
	}
	h2.ReadIdleTimeout = 16 * time.Second
}

type myTripper struct {
	Base http.RoundTripper
}

func (rt myTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	userAgent := fmt.Sprintf("FediBlock/%s (+https://github.com/chronos-tachyon/fediblock/)", Version)
	ctx := req.Context()
	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", userAgent)
	return rt.Base.RoundTrip(req)
}

func cmdPrepareData() {
	switch {
	case flagCredentialsFile == "":
		fmt.Fprintf(os.Stderr, "fatal: missing required flag --credentials-file\n")
		os.Exit(1)
	case flagSheetID == "":
		fmt.Fprintf(os.Stderr, "fatal: missing required flag --sheet-id\n")
		os.Exit(1)
	case flagDataFile == "":
		fmt.Fprintf(os.Stderr, "fatal: missing required flag --data-file\n")
		os.Exit(1)
	}
	if flagSheetName == "" {
		flagSheetName = "Sheet1"
	}

	config := loadServiceAccount(flagCredentialsFile, sheets.SpreadsheetsReadonlyScope)

	ctx := context.Background()
	source := config.TokenSource(ctx)

	var rt http.RoundTripper = &transport
	rt = &oauth2.Transport{Base: rt, Source: source}
	rt = myTripper{Base: rt}
	client := http.Client{Transport: rt}

	srv, err := sheets.NewService(ctx, option.WithHTTPClient(&client))
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to initialize the Google Sheets API: %v\n", err)
		os.Exit(1)
	}

	sheetRange := flagSheetName + "!A2:L"
	resp, err := srv.Spreadsheets.Values.Get(flagSheetID, sheetRange).Do()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to read data from Google Sheets: %v\n", err)
		os.Exit(1)
	}

	var file BlockFile
	file.Spec = BlockFileSpecV1
	file.PublishedAt = time.Now().UTC()
	file.Blocks = make(map[string]BlockItem, len(resp.Values))
	for _, row := range resp.Values {
		instanceDomain := gsheetString(row, 0)
		dateReported := gsheetDate(row, 1)
		dateBlocked := gsheetDate(row, 2)
		isBlocked := gsheetBool(row, 3)
		isRacism := gsheetBool(row, 4)
		isMisogyny := gsheetBool(row, 5)
		isQueerphobia := gsheetBool(row, 6)
		isHarassment := gsheetBool(row, 7)
		isFraud := gsheetBool(row, 8)
		reason := gsheetString(row, 9)
		reportedBy := gsheetString(row, 10)
		receiptsURL := gsheetURL(row, 11)

		_ = reportedBy

		if isBlocked {
			file.Blocks[instanceDomain] = BlockItem{
				DateReported:  dateReported,
				DateBlocked:   dateBlocked,
				IsRacism:      isRacism,
				IsMisogyny:    isMisogyny,
				IsQueerphobia: isQueerphobia,
				IsHarassment:  isHarassment,
				IsFraud:       isFraud,
				Reason:        reason,
				ReceiptsURL:   receiptsURL.String(),
			}
		}
	}

	var buf bytes.Buffer
	e := json.NewEncoder(&buf)
	e.SetIndent("", "  ")
	e.SetEscapeHTML(false)
	err = e.Encode(&file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed encode BlockFile to JSON: %v\n", err)
		os.Exit(1)
	}

	raw := buf.Bytes()
	raw = bytes.TrimSpace(raw)
	raw = bytes.ReplaceAll(raw, []byte{'\n'}, []byte{'\r', '\n'})
	raw = append(raw, '\r', '\n')
	WriteFile(flagDataFile, raw, false)
}

func gsheetString(row []any, index int) string {
	value := row[index]
	switch x := value.(type) {
	case string:
		return x
	default:
		fmt.Fprintf(os.Stderr, "fatal: row[%d] has type %T, expected string\n", index, value)
		os.Exit(1)
		panic(nil)
	}
}

func gsheetDate(row []any, index int) time.Time {
	value := row[index]
	switch x := value.(type) {
	case time.Time:
		return x
	case string:
		t, err := time.ParseInLocation("2006-01-02", x, time.UTC)
		if err != nil {
			t, err = time.Parse(time.RFC3339, x)
		}
		if err != nil {
			t, err = time.Parse(time.RFC3339Nano, x)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "fatal: failed to parse row[%d] = %q as time: %v\n", index, x, err)
			os.Exit(1)
		}
		return t
	default:
		fmt.Fprintf(os.Stderr, "fatal: row[%d] has type %T, expected time.Time or string\n", index, value)
		os.Exit(1)
		panic(nil)
	}
}

func gsheetBool(row []any, index int) bool {
	value := row[index]
	switch x := value.(type) {
	case bool:
		return x
	case int:
		return x != 0
	case int64:
		return x != 0
	case int32:
		return x != 0
	case int16:
		return x != 0
	case int8:
		return x != 0
	case uint:
		return x != 0
	case uintptr:
		return x != 0
	case uint64:
		return x != 0
	case uint32:
		return x != 0
	case uint16:
		return x != 0
	case uint8:
		return x != 0
	case string:
		b, ok := parseBool(x)
		if !ok {
			fmt.Fprintf(os.Stderr, "fatal: failed to parse row[%d] = %q as bool\n", index, x)
			os.Exit(1)
		}
		return b
	default:
		fmt.Fprintf(os.Stderr, "fatal: row[%d] has type %T, expected bool, int, or string\n", index, value)
		os.Exit(1)
		panic(nil)
	}
}

func gsheetURL(row []any, index int) *url.URL {
	value := row[index]
	switch x := value.(type) {
	case *url.URL:
		return x
	case url.URL:
		return &x
	case string:
		u, err := url.Parse(x)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fatal: failed to parse row[%d] = %q as URL: %v\n", index, x, err)
			os.Exit(1)
		}
		return u
	default:
		fmt.Fprintf(os.Stderr, "fatal: row[%d] has type %T, expected *url.URL or string\n", index, value)
		os.Exit(1)
		panic(nil)
	}
}

func parseBool(str string) (value bool, ok bool) {
	str = strings.ToLower(str)
	switch str {
	case "0":
		return false, true
	case "n":
		return false, true
	case "no":
		return false, true
	case "f":
		return false, true
	case "false":
		return false, true
	case "1":
		return true, true
	case "y":
		return true, true
	case "yes":
		return true, true
	case "t":
		return true, true
	case "true":
		return true, true
	default:
		return false, false
	}
}
