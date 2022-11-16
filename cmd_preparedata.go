package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
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
		fmt.Fprintf(os.Stderr, "fatal: missing required flag -K / --credentials-file\n")
		os.Exit(1)
	case flagSheetID == "":
		fmt.Fprintf(os.Stderr, "fatal: missing required flag -H / --sheet-id\n")
		os.Exit(1)
	case flagDataFile == "":
		fmt.Fprintf(os.Stderr, "fatal: missing required flag -d / --data-file\n")
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

	resp, err := srv.Spreadsheets.Values.Get(flagSheetID, flagSheetName).Do()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to read data from Google Sheets: %v\n", err)
		os.Exit(1)
	}

	schema := parseSchema(resp.Values[0])

	var file BlockFile
	file.Spec = BlockFileSpecV1
	file.PublishedAt = time.Now().UTC()
	file.Blocks = make(map[string]Block, len(resp.Values))
	for _, row := range resp.Values[1:] {
		block, err := schema.parseRow(row)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
			os.Exit(1)
		}
		if !block.IsBlock {
			continue
		}
		file.Blocks[block.Domain] = block.Block
	}

	WriteJsonFile(flagDataFile, file, false)
}

type columnSchema []columnID

type columnID byte

const (
	ignoreColumn columnID = iota
	instanceColumn
	dateReportedColumn
	dateBlockedColumn
	reasonColumn
	reporterColumn
	receiptsColumn
	isBlockColumn
	isRacismColumn
	isAntisemitismColumn
	isMisogynyColumn
	isQueerphobiaColumn
	isHarassmentColumn
	isFraudColumn
	isCopyrightColumn
)

type columnNameRule struct {
	ID      columnID
	Pattern *regexp.Regexp
}

var columnNameRules = [...]columnNameRule{
	{instanceColumn, regexp.MustCompile(`^(?i)instance(?:\s+\([^()]*\))?$`)},
	{dateReportedColumn, regexp.MustCompile(`^(?i)date\s+reported(?:\s+\([^()]*\))?$`)},
	{dateBlockedColumn, regexp.MustCompile(`^(?i)date\s+blocked(?:\s+\([^()]*\))?$`)},
	{reasonColumn, regexp.MustCompile(`^(?i)reason(?:\s+\([^()]*\))?$`)},
	{reporterColumn, regexp.MustCompile(`^(?i)reporter(?:\s+url)?(?:\s+\([^()]*\))?$`)},
	{receiptsColumn, regexp.MustCompile(`^(?i)receipts(?:\s+url)?(?:\s+\([^()]*\))?$`)},
	{isBlockColumn, regexp.MustCompile(`^(?i)(?:is\s+)?block\??(?:\s+\([^()]*\))?$`)},
	{isRacismColumn, regexp.MustCompile(`^(?i)(?:is\s+)?racism\??(?:\s+\([^()]*\))?$`)},
	{isAntisemitismColumn, regexp.MustCompile(`^(?i)(?:is\s+)?antisemitism\??(?:\s+\([^()]*\))?$`)},
	{isMisogynyColumn, regexp.MustCompile(`^(?i)(?:is\s+)?misogyny\??(?:\s+\([^()]*\))?$`)},
	{isQueerphobiaColumn, regexp.MustCompile(`^(?i)(?:is\s+)?queerphobia\??(?:\s+\([^()]*\))?$`)},
	{isHarassmentColumn, regexp.MustCompile(`^(?i)(?:is\s+)?harassment\??(?:\s+\([^()]*\))?$`)},
	{isFraudColumn, regexp.MustCompile(`^(?i)(?:is\s+)?fraud\??(?:\s+\([^()]*\))?$`)},
	{isCopyrightColumn, regexp.MustCompile(`^(?i)(?:is\s+)?copyright\??(?:\s+\([^()]*\))?$`)},
}

func parseSchema(row []any) columnSchema {
	cs := make(columnSchema, len(row))
	for index, row := range row {
		name := row.(string)
		id := ignoreColumn
		for _, rule := range columnNameRules {
			if rule.Pattern.MatchString(name) {
				id = rule.ID
				break
			}
		}
		cs[index] = id
	}
	return cs
}

func (cs columnSchema) parseRow(row []any) (block PrivateBlock, err error) {
	for index, value := range row {
		id := ignoreColumn
		if index >= 0 && index < len(cs) {
			id = cs[index]
		}

		switch id {
		case instanceColumn:
			err = gsheetString(&block.Domain, value, index)
		case dateReportedColumn:
			err = gsheetDate(&block.Block.DateReported, value, index)
		case dateBlockedColumn:
			err = gsheetDate(&block.Block.DateBlocked, value, index)
		case reasonColumn:
			err = gsheetString(&block.Block.Reason, value, index)
		case reporterColumn:
			err = gsheetURL(&block.Reporter, value, index)
		case receiptsColumn:
			err = gsheetURL(&block.Receipts, value, index)
		case isBlockColumn:
			err = gsheetBool(&block.IsBlock, value, index)
		case isRacismColumn:
			err = gsheetBool(&block.Block.IsRacism, value, index)
		case isAntisemitismColumn:
			err = gsheetBool(&block.Block.IsAntisemitism, value, index)
		case isMisogynyColumn:
			err = gsheetBool(&block.Block.IsMisogyny, value, index)
		case isQueerphobiaColumn:
			err = gsheetBool(&block.Block.IsQueerphobia, value, index)
		case isHarassmentColumn:
			err = gsheetBool(&block.Block.IsHarassment, value, index)
		case isFraudColumn:
			err = gsheetBool(&block.Block.IsFraud, value, index)
		case isCopyrightColumn:
			err = gsheetBool(&block.Block.IsCopyright, value, index)
		}
		if err != nil {
			break
		}
	}
	if block.Receipts != nil {
		block.Block.ReceiptsURL = block.Receipts.String()
	}
	return block, err
}

func gsheetString(out *string, in any, index int) error {
	switch x := in.(type) {
	case string:
		x = strings.TrimSpace(x)
		*out = x
		return nil
	default:
		return fmt.Errorf("row[%d] has type %T, expected string", index, in)
	}
}

func gsheetDate(out *time.Time, in any, index int) error {
	switch x := in.(type) {
	case *time.Time:
		*out = *x
		return nil
	case time.Time:
		*out = x
		return nil
	case string:
		x = strings.TrimSpace(x)
		if x == "" {
			return nil
		}
		t, err := time.ParseInLocation("2006-01-02", x, time.UTC)
		if err != nil {
			t, err = time.Parse(time.RFC3339, x)
		}
		if err != nil {
			t, err = time.Parse(time.RFC3339Nano, x)
		}
		if err != nil {
			return fmt.Errorf("row[%d] = %q could not be parsed as a date/time: %v\n", index, x, err)
		}
		*out = t
		return nil
	default:
		return fmt.Errorf("row[%d] has type %T, expected time.Time or string", index, in)
	}
}

func gsheetURL(out **url.URL, in any, index int) error {
	switch x := in.(type) {
	case *url.URL:
		*out = x
		return nil
	case url.URL:
		*out = &x
		return nil
	case string:
		x = strings.TrimSpace(x)
		if x == "" {
			return nil
		}
		u, err := url.Parse(x)
		if err != nil {
			return fmt.Errorf("row[%d] = %q could not be parsed as a URL: %v\n", index, x, err)
		}
		*out = u
		return nil
	default:
		return fmt.Errorf("row[%d] has type %T, expected *url.URL or string", index, in)
	}
}

func gsheetBool(out *bool, in any, index int) error {
	switch x := in.(type) {
	case bool:
		*out = x
		return nil
	case int:
		*out = (x != 0)
		return nil
	case int64:
		*out = (x != 0)
		return nil
	case int32:
		*out = (x != 0)
		return nil
	case int16:
		*out = (x != 0)
		return nil
	case int8:
		*out = (x != 0)
		return nil
	case uint:
		*out = (x != 0)
		return nil
	case uintptr:
		*out = (x != 0)
		return nil
	case uint64:
		*out = (x != 0)
		return nil
	case uint32:
		*out = (x != 0)
		return nil
	case uint16:
		*out = (x != 0)
		return nil
	case uint8:
		*out = (x != 0)
		return nil
	case string:
		x = strings.TrimSpace(x)
		if x == "" {
			return nil
		}
		switch strings.ToLower(x) {
		case "0":
			fallthrough
		case "n":
			fallthrough
		case "no":
			fallthrough
		case "f":
			fallthrough
		case "false":
			*out = false
			return nil

		case "1":
			fallthrough
		case "y":
			fallthrough
		case "yes":
			fallthrough
		case "t":
			fallthrough
		case "true":
			*out = true
			return nil

		default:
			return fmt.Errorf("row[%d] = %q could not be parsed as a bool", index, x)
		}
	default:
		return fmt.Errorf("row[%d] has type %T, expected bool, int, or string", index, in)
	}
}
