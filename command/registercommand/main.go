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
	"os/exec"
	"strings"

	getopt "github.com/pborman/getopt/v2"
	"github.com/rs/zerolog"

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

type registerFactory struct {
	command.BaseFactory
}

func (registerFactory) Name() string {
	return "register"
}

func (registerFactory) Description() string {
	return "Registers as an app with the given Mastodon instance, yielding a client token for \"rapidblock apply\"."
}

func (registerFactory) New(dispatcher command.Dispatcher) (*getopt.Set, command.MainFunc) {
	var baseURL string
	var launchBrowser bool

	options := getopt.New()
	options.SetParameters("")
	options.FlagLong(&baseURL, "url", 'u', "URL to the Mastodon instance to register with")
	options.FlagLong(&launchBrowser, "launch-browser", 'x', "launch a web browser using xdg-open?")

	return options, func(ctx context.Context) int {
		if baseURL == "" {
			zerolog.Ctx(ctx).
				Error().
				Msg("missing required flag -u / --url")
			return 1
		}
		return Main(
			ctx,
			launchBrowser,
			baseURL,
			dispatcher.Stdin(),
			dispatcher.Stdout(),
			dispatcher.Stderr(),
		)
	}
}

var Factory command.Factory = registerFactory{}

func Main(
	ctx context.Context,
	launchBrowser bool,
	baseURL string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
) int {
	logger := zerolog.Ctx(ctx).
		With().
		Bool("launchBrowser", launchBrowser).
		Str("baseURL", baseURL).
		Logger()

	u, err := url.Parse(baseURL)
	if err != nil {
		logger.Error().Err(err).Send()
		return 1
	}

	if !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https") {
		logger.Error().
			Str("scheme", u.Scheme).
			Msg("only \"http\" and \"https\" schemes are supported")
		return 1
	}
	if u.User != nil {
		logger.Error().
			Msg("found username/password information in the URL, which is not permitted")
		return 1
	}

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
		logger.Error().
			Str("method", reqMethod).
			Str("url", reqURL).
			Err(err).
			Msg("failed to create HTTP request")
		return 1
	}

	req.Header = make(http.Header, 16)
	req.Header.Set("user-agent", appversion.UserAgent())
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("accept", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		logger.Error().
			Str("method", reqMethod).
			Str("url", reqURL).
			Err(err).
			Msg("HTTP request failed")
		return 1
	}

	respStatus := resp.StatusCode
	respBody, err := io.ReadAll(resp.Body)
	if err2 := resp.Body.Close(); err == nil {
		err = err2
	}
	if err != nil {
		logger.Error().
			Str("method", reqMethod).
			Str("url", reqURL).
			Int("statusCode", respStatus).
			Err(err).
			Msg("I/O error while receiving HTTP response")
		return 1
	}
	if respStatus != http.StatusOK {
		logger.Error().
			Str("method", reqMethod).
			Str("url", reqURL).
			Int("statusCode", respStatus).
			Msg("server returned unexpected HTTP status code")
		httpclient.WriteDebug(stderr, resp.Header, respBody)
		return 1
	}

	var arr appRegisterResponse
	d := json.NewDecoder(bytes.NewReader(respBody))
	d.UseNumber()
	err = d.Decode(&arr)
	if err != nil {
		logger.Error().
			Str("method", reqMethod).
			Str("url", reqURL).
			Int("statusCode", respStatus).
			Err(err).
			Msg("failed to parse HTTP response body as JSON")
		httpclient.WriteDebug(stderr, resp.Header, respBody)
		return 1
	}

	browseURL := u.JoinPath("/oauth/authorize")
	q = make(url.Values, 16)
	q.Set("client_id", arr.ClientID)
	q.Set("scope", OAuthScopes)
	q.Set("redirect_uri", OAuthRedirectURL)
	q.Set("response_type", "code")
	browseURL.RawQuery = q.Encode()
	browseURLString := browseURL.String()

	_, err = io.WriteString(stdout, fmt.Sprintf("%s\n", browseURLString))
	if err != nil {
		logger.Error().
			Err(err).
			Msg("I/O error while writing to stdout")
		return 1
	}

	if launchBrowser {
		//nolint:gosec
		cmd := exec.CommandContext(ctx, "xdg-open", browseURLString)
		err = cmd.Run()
		if err != nil {
			logger.Warn().
				Strs("args", cmd.Args).
				Err(err).
				Msg("failed to launch web browser")
		}
	}

	_, err = io.WriteString(stdout, "Code: ")
	if err != nil {
		logger.Error().
			Err(err).
			Msg("I/O error while writing to stdout")
		return 1
	}

	stdinBuffered := bufio.NewReader(stdin)
	line, err := stdinBuffered.ReadString('\n')
	if err != nil && err != io.EOF {
		logger.Error().
			Err(err).
			Msg("I/O error while reading from stdin")
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
		logger.Error().
			Str("method", reqMethod).
			Str("url", reqURL).
			Err(err).
			Msg("failed to create HTTP request")
		return 1
	}

	req.Header = make(http.Header, 16)
	req.Header.Set("user-agent", appversion.UserAgent())
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("accept", "application/json")

	resp, err = c.Do(req)
	if err != nil {
		logger.Error().
			Str("method", reqMethod).
			Str("url", reqURL).
			Err(err).
			Msg("HTTP request failed")
		return 1
	}

	respStatus = resp.StatusCode
	respBody, err = io.ReadAll(resp.Body)
	if err2 := resp.Body.Close(); err == nil {
		err = err2
	}
	if err != nil {
		logger.Error().
			Str("method", reqMethod).
			Str("url", reqURL).
			Int("statusCode", respStatus).
			Err(err).
			Msg("I/O error while receiving HTTP response")
		return 1
	}
	if respStatus != http.StatusOK {
		logger.Error().
			Str("method", reqMethod).
			Str("url", reqURL).
			Int("statusCode", respStatus).
			Msg("server returned unexpected HTTP status code")
		httpclient.WriteDebug(stderr, resp.Header, respBody)
		return 1
	}

	var otr oauthTokenResponse
	d = json.NewDecoder(bytes.NewReader(respBody))
	d.UseNumber()
	err = d.Decode(&otr)
	if err != nil {
		logger.Error().
			Str("method", reqMethod).
			Str("url", reqURL).
			Int("statusCode", respStatus).
			Err(err).
			Msg("failed to parse HTTP response body as JSON")
		httpclient.WriteDebug(stderr, resp.Header, respBody)
		return 1
	}

	if otr.TokenType != "Bearer" {
		logger.Error().
			Str("method", reqMethod).
			Str("url", reqURL).
			Int("statusCode", respStatus).
			Str("tokenType", otr.TokenType).
			Msg("unknown token type; only \"Bearer\" is supported")
		httpclient.WriteDebug(stderr, resp.Header, respBody)
		return 1
	}

	_, err = io.WriteString(
		stdout,
		fmt.Sprintf(
			"app_id: %d\nclient_id: %s\nclient_secret: %s\nclient_token: %s\nclient_token_type: %s\n",
			arr.ID,
			arr.ClientID,
			arr.ClientSecret,
			otr.AccessToken,
			otr.TokenType,
		),
	)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("I/O error while writing to stdout")
		return 1
	}
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
