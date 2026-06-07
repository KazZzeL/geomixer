package fetcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var errBadStatus = errors.New("wrong response code for")

func URL(ctx context.Context, hc *http.Client, ht time.Duration, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, ht)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w %s: %s", errBadStatus, url, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return body, nil
}
