package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func Do[T any](
	ctx context.Context,
	client *http.Client,
	reqMethod string,
	reqURL *url.URL,
	reqBody io.ReadCloser,
	reqFn func(*http.Request),
	okFn func(int) bool,
	out *T,
) error {
	reqURLString := reqURL.String()

	if reqBody == nil {
		reqBody = http.NoBody
	}

	req, err := http.NewRequestWithContext(ctx, reqMethod, reqURLString, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: method=%q url=%q: %w", reqMethod, reqURLString, err)
	}

	if reqFn != nil {
		reqFn(req)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: method=%q url=%q: %w", reqMethod, reqURLString, err)
	}

	respStatus := resp.StatusCode
	respBody, err := io.ReadAll(resp.Body)
	err2 := resp.Body.Close()
	if err == nil {
		err = err2
	}
	if err != nil {
		return fmt.Errorf("I/O error while reading HTTP response body: method=%q url=%q %w", reqMethod, reqURLString, err)
	}
	if okFn != nil && !okFn(respStatus) {
		return fmt.Errorf("HTTP response has unexpected status code: method=%q url=%q status=%03d", reqMethod, reqURLString, respStatus)
	}

	var tmp T
	err = json.Unmarshal(respBody, &tmp)
	if err != nil {
		return fmt.Errorf("failed to decode HTTP response body as JSON: method=%q url=%q status=%03d: %w", reqMethod, reqURLString, respStatus, err)
	}

	*out = tmp
	return nil
}
