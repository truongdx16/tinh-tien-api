package notification

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
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Message is what a push shows, plus data for the app (endpoint to open).
type Message struct {
	Title string
	Body  string
	Data  map[string]string
}

// Sender delivers a message to device tokens. It returns the tokens FCM
// reports as no longer valid (app uninstalled, token rotated) so they can be
// deleted; err is for failures that are not about a single token.
type Sender interface {
	Send(ctx context.Context, tokens []string, msg Message) (sent int, invalid []string, err error)
	Enabled() bool
}

// NoopSender is used when no Firebase credentials are configured:
// notifications are still stored (and listed in the app), not pushed.
type NoopSender struct{}

func (NoopSender) Send(context.Context, []string, Message) (int, []string, error) { return 0, nil, nil }
func (NoopSender) Enabled() bool                                                  { return false }

// ServiceAccount is the part of a Firebase service-account JSON key used to
// call FCM.
type ServiceAccount struct {
	ProjectID   string `json:"project_id"`
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}

// ParseServiceAccount reads a service-account JSON key.
func ParseServiceAccount(data []byte) (*ServiceAccount, error) {
	var sa ServiceAccount
	if err := json.Unmarshal(data, &sa); err != nil {
		return nil, fmt.Errorf("parse service account: %w", err)
	}
	if sa.ProjectID == "" || sa.ClientEmail == "" || sa.PrivateKey == "" {
		return nil, errors.New("service account: project_id, client_email and private_key are required")
	}
	if sa.TokenURI == "" {
		sa.TokenURI = "https://oauth2.googleapis.com/token"
	}
	return &sa, nil
}

// LoadServiceAccountFile reads a service-account JSON key file.
func LoadServiceAccountFile(path string) (*ServiceAccount, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read service account: %w", err)
	}
	return ParseServiceAccount(data)
}

const fcmScope = "https://www.googleapis.com/auth/firebase.messaging"

// FCMSender sends through the FCM HTTP v1 API, authenticating with a
// service account (OAuth2 JWT bearer flow, token cached until it expires).
type FCMSender struct {
	sa     *ServiceAccount
	client *http.Client
	// BaseURL of the FCM API; overridden in tests.
	BaseURL string
	// AndroidChannel is the notification channel on Android (must match the
	// channel the app creates).
	AndroidChannel string

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

func NewFCMSender(sa *ServiceAccount) *FCMSender {
	return &FCMSender{
		sa:             sa,
		client:         &http.Client{Timeout: 15 * time.Second},
		BaseURL:        "https://fcm.googleapis.com",
		AndroidChannel: "planting_schedule_channel",
	}
}

func (s *FCMSender) Enabled() bool { return true }

func (s *FCMSender) token(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.accessToken != "" && time.Until(s.expiresAt) > time.Minute {
		return s.accessToken, nil
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(s.sa.PrivateKey))
	if err != nil {
		return "", fmt.Errorf("service account private key: %w", err)
	}
	now := time.Now()
	assertion, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":   s.sa.ClientEmail,
		"scope": fcmScope,
		"aud":   s.sa.TokenURI,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}).SignedString(key)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}

	form := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {assertion},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.sa.TokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("oauth token: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("oauth token: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &out); err != nil || out.AccessToken == "" {
		return "", fmt.Errorf("oauth token: unexpected response")
	}
	s.accessToken = out.AccessToken
	s.expiresAt = now.Add(time.Duration(out.ExpiresIn) * time.Second)
	return s.accessToken, nil
}

type fcmMessage struct {
	Message fcmPayload `json:"message"`
}

type fcmPayload struct {
	Token        string            `json:"token"`
	Notification map[string]string `json:"notification"`
	Data         map[string]string `json:"data,omitempty"`
	Android      map[string]any    `json:"android"`
	APNS         map[string]any    `json:"apns"`
}

// Send pushes msg to every token (FCM v1 takes one token per request).
func (s *FCMSender) Send(ctx context.Context, tokens []string, msg Message) (int, []string, error) {
	if len(tokens) == 0 {
		return 0, nil, nil
	}
	access, err := s.token(ctx)
	if err != nil {
		return 0, nil, err
	}
	endpoint := fmt.Sprintf("%s/v1/projects/%s/messages:send", strings.TrimRight(s.BaseURL, "/"), s.sa.ProjectID)

	sent := 0
	var invalid []string
	var lastErr error
	for _, token := range tokens {
		body, _ := json.Marshal(fcmMessage{Message: fcmPayload{
			Token:        token,
			Notification: map[string]string{"title": msg.Title, "body": msg.Body},
			Data:         msg.Data,
			Android: map[string]any{
				"priority":     "high",
				"notification": map[string]any{"channel_id": s.AndroidChannel},
			},
			APNS: map[string]any{
				"payload": map[string]any{"aps": map[string]any{"sound": "default"}},
			},
		}})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return sent, invalid, err
		}
		req.Header.Set("Authorization", "Bearer "+access)
		req.Header.Set("Content-Type", "application/json")
		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		resp.Body.Close()
		switch {
		case resp.StatusCode == http.StatusOK:
			sent++
		case isInvalidToken(resp.StatusCode, respBody):
			invalid = append(invalid, token)
		default:
			lastErr = fmt.Errorf("fcm send: %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
		}
	}
	return sent, invalid, lastErr
}

// isInvalidToken: the token will never work again (FCM UNREGISTERED, or a
// malformed token reported as INVALID_ARGUMENT on the token field).
func isInvalidToken(status int, body []byte) bool {
	if status != http.StatusNotFound && status != http.StatusBadRequest {
		return false
	}
	var e struct {
		Error struct {
			Status  string `json:"status"`
			Message string `json:"message"`
			Details []struct {
				ErrorCode string `json:"errorCode"`
			} `json:"details"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &e) != nil {
		return false
	}
	for _, d := range e.Error.Details {
		if d.ErrorCode == "UNREGISTERED" {
			return true
		}
	}
	if e.Error.Status == "NOT_FOUND" {
		return true
	}
	return e.Error.Status == "INVALID_ARGUMENT" &&
		strings.Contains(strings.ToLower(e.Error.Message), "registration token")
}
