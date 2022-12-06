package mastodon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/chronos-tachyon/rapidblock/blockapply"
	"github.com/chronos-tachyon/rapidblock/blockfile"
	"github.com/chronos-tachyon/rapidblock/internal/httpclient"
)

var errNotImplemented = errors.New("not implemented")

func ApplyREST(ctx context.Context, server blockapply.Server, file blockfile.BlockFile) (stats blockapply.Stats, err error) {
	u, err := url.Parse(server.URI)
	if err != nil {
		return stats, fmt.Errorf("failed to parse base URL: %q: %w", server.URI, err)
	}

	h := make(http.Header, 2)
	h.Set("user-agent", httpclient.UserAgent)
	h.Set("authorization", fmt.Sprintf("Bearer %s", server.ClientToken))

	var rt http.RoundTripper = httpclient.Transport()
	rt = &httpclient.RequestHeaderRoundTripper{Next: rt, Header: h}
	c := httpclient.Client(rt)
	applier := &RESTApplier{Client: c, BaseURL: u}
	return Apply(ctx, applier, file)
}

type RESTApplier struct {
	Client  *http.Client
	BaseURL *url.URL
}

func (applier *RESTApplier) Query(ctx context.Context, out map[string]DomainBlock) error {
	method := http.MethodGet
	urlstr := applier.BaseURL.JoinPath("/api/v1/admin/domain_blocks").String()

	looping := true
	for looping {
		req, err := http.NewRequestWithContext(ctx, method, urlstr, http.NoBody)
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %q, %q: %w", method, urlstr, err)
		}

		req.Header = make(http.Header, 16)
		req.Header.Set("accept", "application/json")

		resp, err := applier.Client.Do(req)
		if err != nil {
			return fmt.Errorf("HTTP request failed: %q, %q: %w", method, urlstr, err)
		}

		respStatus := resp.StatusCode
		respBody, err := io.ReadAll(resp.Body)
		if err2 := resp.Body.Close(); err == nil {
			err = err2
		}
		if err != nil {
			return fmt.Errorf("I/O error while receiving HTTP response: %q, %q, %03d: %w", method, urlstr, respStatus, err)
		}
		if respStatus != http.StatusOK {
			return fmt.Errorf("server returned unexpected HTTP status code %03d", respStatus)
		}

		var blocks []DomainBlock
		d := json.NewDecoder(bytes.NewReader(respBody))
		err = d.Decode(&blocks)
		if err != nil {
			return fmt.Errorf("failed to parse HTTP response body as JSON: %q, %q, %03d: %w", method, urlstr, respStatus, err)
		}

		for _, block := range blocks {
			out[block.Domain] = block
		}

		links, err := extractLinks(resp.Header)
		if err != nil {
			return fmt.Errorf("failed to extract Link headers from HTTP response: %w", err)
		}

		looping = false
		for _, item := range links {
			if item.Rel == "next" {
				urlstr = item.URL.String()
				looping = true
				break
			}
		}
	}
	return nil
}

func (applier *RESTApplier) Insert(ctx context.Context, block DomainBlock) error {
	method := http.MethodPost
	urlstr := applier.BaseURL.JoinPath("/api/v1/admin/domain_blocks").String()

	q := make(url.Values, 16)
	q.Set("domain", block.Domain)
	setQuerySeverity(q, "severity", block.Severity)
	setQueryNullString(q, "private_comment", block.PrivateComment)
	setQueryNullString(q, "public_comment", block.PublicComment)
	setQueryBool(q, "reject_media", block.RejectMedia)
	setQueryBool(q, "reject_reports", block.RejectReports)
	setQueryBool(q, "obfuscate", block.Obfuscate)
	bodyString := q.Encode()
	reqBody := io.NopCloser(strings.NewReader(bodyString))

	req, err := http.NewRequestWithContext(ctx, method, urlstr, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %q, %q: %w", method, urlstr, err)
	}

	req.ContentLength = int64(len(bodyString))
	req.Header = make(http.Header, 16)
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("accept", "application/json")

	resp, err := applier.Client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %q, %q: %w", method, urlstr, err)
	}

	respStatus := resp.StatusCode
	respBody, err := io.ReadAll(resp.Body)
	if err2 := resp.Body.Close(); err == nil {
		err = err2
	}
	if err != nil {
		return fmt.Errorf("I/O error while receiving HTTP response: %q, %q, %03d: %w", method, urlstr, respStatus, err)
	}
	if respStatus != http.StatusOK {
		return fmt.Errorf("server returned unexpected HTTP status code %03d", respStatus)
	}

	var result DomainBlock
	d := json.NewDecoder(bytes.NewReader(respBody))
	err = d.Decode(&result)
	if err != nil {
		return fmt.Errorf("failed to parse HTTP response body as JSON: %q, %q, %03d: %w", method, urlstr, respStatus, err)
	}

	fmt.Fprintf(os.Stderr, "debug: %+v\n", result)
	return nil
}

func (applier *RESTApplier) Update(ctx context.Context, block DomainBlock) error {
	idStr := block.ID.String()
	method := http.MethodPut
	urlstr := applier.BaseURL.JoinPath("/api/v1/admin/domain_blocks", idStr).String()

	q := make(url.Values, 16)
	setQuerySeverity(q, "severity", block.Severity)
	setQueryNullString(q, "private_comment", block.PrivateComment)
	setQueryNullString(q, "public_comment", block.PublicComment)
	setQueryBool(q, "reject_media", block.RejectMedia)
	setQueryBool(q, "reject_reports", block.RejectReports)
	setQueryBool(q, "obfuscate", block.Obfuscate)
	bodyString := q.Encode()
	reqBody := io.NopCloser(strings.NewReader(bodyString))

	req, err := http.NewRequestWithContext(ctx, method, urlstr, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %q, %q: %w", method, urlstr, err)
	}

	req.ContentLength = int64(len(bodyString))
	req.Header = make(http.Header, 16)
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("accept", "application/json")

	resp, err := applier.Client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %q, %q: %w", method, urlstr, err)
	}

	respStatus := resp.StatusCode
	respBody, err := io.ReadAll(resp.Body)
	if err2 := resp.Body.Close(); err == nil {
		err = err2
	}
	if err != nil {
		return fmt.Errorf("I/O error while receiving HTTP response: %q, %q, %03d: %w", method, urlstr, respStatus, err)
	}
	if respStatus != http.StatusOK {
		return fmt.Errorf("server returned unexpected HTTP status code %03d", respStatus)
	}

	var result DomainBlock
	d := json.NewDecoder(bytes.NewReader(respBody))
	err = d.Decode(&result)
	if err != nil {
		return fmt.Errorf("failed to parse HTTP response body as JSON: %q, %q, %03d: %w", method, urlstr, respStatus, err)
	}

	fmt.Fprintf(os.Stderr, "debug: %+v\n", result)
	return nil
}

func (applier *RESTApplier) Delete(ctx context.Context, block DomainBlock) error {
	idStr := block.ID.String()
	method := http.MethodDelete
	urlstr := applier.BaseURL.JoinPath("/api/v1/admin/domain_blocks", idStr).String()

	req, err := http.NewRequestWithContext(ctx, method, urlstr, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %q, %q: %w", method, urlstr, err)
	}

	req.Header = make(http.Header, 16)
	req.Header.Set("accept", "application/json")

	resp, err := applier.Client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %q, %q: %w", method, urlstr, err)
	}

	respStatus := resp.StatusCode
	respBody, err := io.ReadAll(resp.Body)
	if err2 := resp.Body.Close(); err == nil {
		err = err2
	}
	if err != nil {
		return fmt.Errorf("I/O error while receiving HTTP response: %q, %q, %03d: %w", method, urlstr, respStatus, err)
	}
	if respStatus != http.StatusOK {
		return fmt.Errorf("server returned unexpected HTTP status code %03d", respStatus)
	}

	var empty struct{}
	d := json.NewDecoder(bytes.NewReader(respBody))
	err = d.Decode(&empty)
	if err != nil {
		return fmt.Errorf("failed to parse HTTP response body as JSON: %q, %q, %03d: %w", method, urlstr, respStatus, err)
	}

	return nil
}

var _ Applier = (*RESTApplier)(nil)

type link struct {
	URL            *url.URL
	Rel            string
	Type           string
	Lang           string
	Title          string
	As             string
	Media          string
	Sizes          string
	ImageSizes     string
	ImageSrcSet    string
	Integrity      string
	ReferrerPolicy string
	CrossOrigin    string
	Prefetch       string
	Blocking       string
}

func extractLinks(header http.Header) ([]link, error) {
	values := header.Values("link")
	if len(values) == 0 {
		return nil, nil
	}

	n := len(values)
	if n < 64 {
		n = 64
	}

	type state byte
	const (
		stateInitial state = iota
		stateURL
		stateWantParam
		stateSemicolon
		stateParamName
		stateEqual
		stateParamValueUnquoted
		stateParamValueQuoted
		stateParamValueQuotedBS
	)

	var buf bytes.Buffer
	var partial link
	var paramName string
	var s state

	flushURL := func() error {
		str := buf.String()
		buf.Reset()
		u, err := url.Parse(str)
		if err != nil {
			return fmt.Errorf("failed to parse URL: %q: %w", str, err)
		}
		partial.URL = u
		return nil
	}

	flushParamName := func() {
		paramName = buf.String()
		buf.Reset()
	}

	flushParamValue := func() {
		paramValue := buf.String()
		buf.Reset()
		switch strings.ToLower(paramName) {
		case "rel":
			partial.Rel = paramValue
		case "type":
			partial.Type = paramValue
		case "lang":
			fallthrough
		case "hreflang":
			partial.Lang = paramValue
		case "title":
			partial.Title = paramValue
		case "as":
			partial.As = paramValue
		case "media":
			partial.Media = paramValue
		case "sizes":
			partial.Sizes = paramValue
		case "imagesizes":
			partial.ImageSizes = paramValue
		case "imagesrcset":
			partial.ImageSrcSet = paramValue
		case "integrity":
			partial.Integrity = paramValue
		case "refererpolicy":
			fallthrough
		case "referrerpolicy":
			partial.ReferrerPolicy = paramValue
		case "crossorigin":
			partial.CrossOrigin = paramValue
		case "prefetch":
			partial.Prefetch = paramValue
		case "blocking":
			partial.Blocking = paramValue
		default:
			fmt.Fprintf(os.Stderr, "warn: unknown Link header param %q=%q\n", paramName, paramValue)
		}
	}

	out := make([]link, 0, n)
	flushLink := func() {
		out = append(out, partial)
		partial = link{}
	}

	for _, value := range header.Values("link") {
		buf.Reset()
		partial = link{}
		s = stateInitial
		for _, ch := range value {
			cls := classify(ch)

			switch {
			case s == stateInitial && cls == clsSpace:
				// pass
			case s == stateInitial && cls == clsComma:
				// pass
			case s == stateInitial && cls == clsLT:
				s = stateURL
			case s == stateInitial:
				return nil, fmt.Errorf("unexpected character in Link header: expected '<', got %q in %q", ch, value)

			case s == stateURL && cls == clsGT:
				err := flushURL()
				if err != nil {
					return nil, err
				}
				s = stateWantParam
			case s == stateURL:
				buf.WriteRune(ch)

			case s == stateWantParam && cls == clsComma:
				flushLink()
				s = stateInitial
			case s == stateWantParam && cls == clsSemicolon:
				s = stateSemicolon
			case s == stateWantParam && cls == clsSpace:
				// pass
			case s == stateWantParam:
				return nil, fmt.Errorf("unexpected character in Link header: expected ';' or ',', got %q in %q", ch, value)

			case s == stateSemicolon && cls == clsSemicolon:
				// pass
			case s == stateSemicolon && cls == clsSpace:
				// pass
			case s == stateSemicolon && cls == clsWord:
				buf.WriteRune(ch)
				s = stateParamName
			case s == stateSemicolon:
				return nil, fmt.Errorf("unexpected character in Link header: expected param name, got %q in %q", ch, value)

			case s == stateParamName && cls == clsComma:
				flushParamName()
				flushParamValue()
				flushLink()
				s = stateInitial
			case s == stateParamName && cls == clsSemicolon:
				flushParamName()
				flushParamValue()
				s = stateSemicolon
			case s == stateParamName && cls == clsEQ:
				flushParamName()
				s = stateEqual
			case s == stateParamName && cls == clsWord:
				buf.WriteRune(ch)
			case s == stateParamName:
				return nil, fmt.Errorf("unexpected character in Link header: expected param name or '=', got %q in %q", ch, value)

			case s == stateEqual && cls == clsDQ:
				s = stateParamValueQuoted
			case s == stateEqual && cls == clsWord:
				buf.WriteRune(ch)
				s = stateParamValueUnquoted
			case s == stateEqual:
				return nil, fmt.Errorf("unexpected character in Link header: expected param value, got %q in %q", ch, value)

			case s == stateParamValueUnquoted && cls == clsComma:
				flushParamValue()
				flushLink()
				s = stateInitial
			case s == stateParamValueUnquoted && cls == clsSemicolon:
				flushParamValue()
				s = stateSemicolon
			case s == stateParamValueUnquoted && cls == clsSpace:
				flushParamValue()
				s = stateWantParam
			case s == stateParamValueUnquoted && cls == clsWord:
				buf.WriteRune(ch)
			case s == stateParamValueUnquoted:
				return nil, fmt.Errorf("unexpected character in Link header: expected param value, got %q in %q", ch, value)

			case s == stateParamValueQuoted && cls == clsDQ:
				flushParamValue()
				s = stateWantParam
			case s == stateParamValueQuoted && cls == clsBS:
				s = stateParamValueQuotedBS
			case s == stateParamValueQuoted:
				buf.WriteRune(ch)

			case s == stateParamValueQuotedBS:
				buf.WriteRune(ch)
				s = stateParamValueQuoted
			}
		}
	}
	return out, nil
}

type classification byte

const (
	clsOther classification = iota
	clsComma
	clsSemicolon
	clsDQ
	clsBS
	clsEQ
	clsLT
	clsGT
	clsWord
	clsSpace
)

func classify(ch rune) classification {
	switch {
	case ch == ',':
		return clsComma
	case ch == ';':
		return clsSemicolon
	case ch == '"':
		return clsDQ
	case ch == '\\':
		return clsBS
	case ch == '=':
		return clsEQ
	case ch == '<':
		return clsLT
	case ch == '>':
		return clsGT
	case ch == '-':
		return clsWord
	case ch == '_':
		return clsWord
	case unicode.IsLetter(ch):
		return clsWord
	case unicode.IsDigit(ch):
		return clsWord
	case unicode.IsSpace(ch):
		return clsSpace
	default:
		return clsOther
	}
}

func setQueryBool(q url.Values, name string, value bool) {
	q.Set(name, strconv.FormatBool(value))
}

func setQuerySeverity(q url.Values, name string, value Severity) {
	q.Set(name, value.String())
}

func setQueryNullString(q url.Values, name string, value NullString) {
	if value.IsValid {
		q.Set(name, value.StringValue)
	} else {
		q.Del(name)
	}
}

func init() {
	blockapply.SetFunc(blockapply.ModeMastodon4xREST, ApplyREST)
}
