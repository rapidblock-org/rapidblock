package registercommand

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	getopt "github.com/pborman/getopt/v2"

	"github.com/chronos-tachyon/rapidblock/command"
	"github.com/chronos-tachyon/rapidblock/internal/appversion"
	"github.com/chronos-tachyon/rapidblock/internal/httpclient"
	"github.com/chronos-tachyon/rapidblock/mastodon"
)

const (
	AppName          = "RapidBlock"
	AppURL           = "https://rapidblock.org/"
	OAuthRedirectURL = "urn:ietf:wg:oauth:2.0:oob"

	// TODO: reduce scopes to admin:{read|write}:domain_blocks for Mastodon versions that support it
	OAuthScopes = "admin:read admin:write"
)

var Factory command.FactoryFunc = func() command.Command {
	var baseURL string
	var launchBrowser bool

	options := getopt.New()
	options.SetParameters("")
	options.FlagLong(&baseURL, "url", 'u', "URL to the Mastodon instance to register with")
	options.FlagLong(&launchBrowser, "launch-browser", 'x', "launch a web browser using xdg-open?")

	return command.Command{
		Name:        "register",
		Description: "Registers as an app with the given Mastodon instance, yielding a client token for \"rapidblock apply\".",
		Options:     options,
		Main: func() int {
			if baseURL == "" {
				fmt.Fprintf(os.Stderr, "fatal: missing required flag -u / --url\n")
				return 1
			}
			return Main(launchBrowser, baseURL)
		},
	}
}

func Main(launchBrowser bool, baseURL string) int {
	u, err := url.Parse(baseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		return 1
	}

	if !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https") {
		fmt.Fprintf(os.Stderr, "fatal: only \"http\" and \"https\" schemes are supported, not %q\n", u.Scheme)
		return 1
	}
	if u.User != nil {
		fmt.Fprintf(os.Stderr, "fatal: found username/password information in the URL, which is not permitted\n\turl = %q\n", u.Redacted())
		return 1
	}

	ctx := context.Background()
	c := httpclient.Client(httpclient.Transport())

	reqMethod := http.MethodPost
	reqURL := u.JoinPath("/api/v1/apps").String()

	q := make(url.Values, 8)
	q.Set("client_name", AppName)
	q.Set("redirect_uris", OAuthRedirectURL)
	q.Set("scopes", OAuthScopes)
	q.Set("website", AppURL)
	reqBody := io.NopCloser(strings.NewReader(q.Encode()))

	req, err := http.NewRequestWithContext(ctx, reqMethod, reqURL, reqBody)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to create HTTP request: %q, %q: %v\n", reqMethod, reqURL, err)
		return 1
	}

	req.Header = make(http.Header, 16)
	req.Header.Set("user-agent", appversion.UserAgent())
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("accept", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: HTTP request failed: %q, %q: %v\n", reqMethod, reqURL, err)
		return 1
	}

	respStatus := resp.StatusCode
	respBody, err := io.ReadAll(resp.Body)
	if err2 := resp.Body.Close(); err == nil {
		err = err2
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: I/O error while receiving HTTP response: %q, %q, %03d: %v\n", reqMethod, reqURL, respStatus, err)
		return 1
	}
	if respStatus != http.StatusOK {
		fmt.Fprintf(os.Stderr, "fatal: server returned unexpected HTTP status code %03d\n", respStatus)
		httpclient.WriteDebug(os.Stderr, resp.Header, respBody)
		return 1
	}

	var arr appRegisterResponse
	d := json.NewDecoder(bytes.NewReader(respBody))
	d.UseNumber()
	err = d.Decode(&arr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to parse HTTP response body as JSON: %v\n", err)
		httpclient.WriteDebug(os.Stderr, resp.Header, respBody)
		return 1
	}

	browseURL := u.JoinPath("/oauth/authorize")
	q = make(url.Values, 16)
	q.Set("client_id", arr.ClientID)
	q.Set("scope", OAuthScopes)
	q.Set("redirect_uri", OAuthRedirectURL)
	q.Set("response_type", "code")
	browseURL.RawQuery = q.Encode()
	fmt.Printf("%s\n", browseURL.String())
	if launchBrowser {
		//nolint:gosec
		cmd := exec.CommandContext(ctx, "xdg-open", browseURL.String())
		err = cmd.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: failed to launch web browser: %q: %v\n", cmd.Args, err)
		}
	}

	os.Stdout.Write([]byte("Code: "))
	stdin := bufio.NewReader(os.Stdin)
	line, err := stdin.ReadString('\n')
	if err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "fatal: failed to read code from stdin: %v\n", err)
		return 1
	}
	oauthCode := strings.TrimSpace(line)

	reqMethod = http.MethodPost
	reqURL = u.JoinPath("/oauth/token").String()

	q = make(url.Values, 8)
	q.Set("client_id", arr.ClientID)
	q.Set("client_secret", arr.ClientSecret)
	q.Set("scope", OAuthScopes)
	q.Set("redirect_uri", OAuthRedirectURL)
	q.Set("grant_type", "authorization_code")
	q.Set("code", oauthCode)
	reqBody = io.NopCloser(strings.NewReader(q.Encode()))

	req, err = http.NewRequestWithContext(ctx, reqMethod, reqURL, reqBody)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to create HTTP request: %q, %q: %v\n", reqMethod, reqURL, err)
		return 1
	}

	req.Header = make(http.Header, 16)
	req.Header.Set("user-agent", appversion.UserAgent())
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("accept", "application/json")

	resp, err = c.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: HTTP request failed: %q, %q: %v\n", reqMethod, reqURL, err)
		return 1
	}

	respStatus = resp.StatusCode
	respBody, err = io.ReadAll(resp.Body)
	if err2 := resp.Body.Close(); err == nil {
		err = err2
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: I/O error while receiving HTTP response: %q, %q, %03d: %v\n", reqMethod, reqURL, respStatus, err)
		return 1
	}
	if respStatus != http.StatusOK {
		fmt.Fprintf(os.Stderr, "fatal: server returned unexpected HTTP status code %03d\n", respStatus)
		httpclient.WriteDebug(os.Stderr, resp.Header, respBody)
		return 1
	}

	var otr oauthTokenResponse
	d = json.NewDecoder(bytes.NewReader(respBody))
	d.UseNumber()
	err = d.Decode(&otr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: failed to parse HTTP response body as JSON: %v\n", err)
		httpclient.WriteDebug(os.Stderr, resp.Header, respBody)
		return 1
	}

	if otr.TokenType != "Bearer" {
		fmt.Fprintf(os.Stdout, "fatal: don't know about token_type %q\n", otr.TokenType)
		httpclient.WriteDebug(os.Stderr, resp.Header, respBody)
		return 1
	}

	fmt.Fprintf(os.Stdout, "app_id: %d\nclient_id: %s\nclient_secret: %s\nclient_token: %s\nclient_token_type: %s\n", arr.ID, arr.ClientID, arr.ClientSecret, otr.AccessToken, otr.TokenType)
	return 0
}

type appRegisterResponse struct {
	ID           mastodon.StringableU64 `json:"id"`
	Name         string                 `json:"name"`
	Website      string                 `json:"website"`
	RedirectURI  string                 `json:"redirect_uri"`
	ClientID     string                 `json:"client_id"`
	ClientSecret string                 `json:"client_secret"`
	VapidKey     string                 `json:"vapid_key"`
}

type oauthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	CreatedAt   int64  `json:"created_at"`
}
