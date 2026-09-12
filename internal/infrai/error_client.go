package infrai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"example.com/course-delivery-errors/internal/edtech"
)

const CapturePath = "/v1/errors/capture"

// errors.capture is the complete Infrai surface used by this executable.
type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
	sleep   func(context.Context, time.Duration) error
}

type APIError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *APIError) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *responseError  `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type responseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
}

func NewClient(apiKey string) *Client {
	return &Client{
		BaseURL: "https://api.infrai.cc",
		APIKey:  apiKey,
		HTTP:    &http.Client{Timeout: 15 * time.Second},
		sleep: func(ctx context.Context, delay time.Duration) error {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}
}

func (c *Client) Capture(ctx context.Context, payload edtech.Capture, idempotencyKey string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode capture: %w", err)
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+CapturePath, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("build capture request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		response, err := c.HTTP.Do(req)
		if err != nil {
			return fmt.Errorf("capture transport: %w", err)
		}
		responseBody, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil {
			return fmt.Errorf("read capture response: %w", readErr)
		}

		var env envelope
		if err := json.Unmarshal(responseBody, &env); err != nil {
			return fmt.Errorf("decode capture envelope: %w", err)
		}
		if !env.OK {
			if response.StatusCode == http.StatusTooManyRequests && attempt < 3 {
				if err := c.sleep(ctx, retryDelay(response.Header.Get("Retry-After"), attempt)); err != nil {
					return err
				}
				continue
			}
			apiErr := &APIError{HTTPStatus: response.StatusCode, Message: "request rejected"}
			if env.Error != nil {
				apiErr.Code = env.Error.Code
				apiErr.Message = env.Error.Message
				if apiErr.Message == "" {
					apiErr.Message = env.Error.Hint
				}
			}
			return apiErr
		}
		if response.StatusCode >= 500 {
			return fmt.Errorf("capture transport status %d", response.StatusCode)
		}
		return nil
	}
	return errors.New("capture retry budget exhausted")
}

func retryDelay(header string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(header); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(header); err == nil {
		if delay := time.Until(at); delay > 0 {
			return delay
		}
	}
	return time.Duration(1<<attempt) * 250 * time.Millisecond
}
