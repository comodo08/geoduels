package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

const turnstileSiteverifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
const guestTurnstileAction = "guest_signup"

type turnstileSiteverifyResponse struct {
	Success    bool     `json:"success"`
	Hostname   string   `json:"hostname"`
	Action     string   `json:"action"`
	ErrorCodes []string `json:"error-codes"`
}

func (a *api) verifyGuestTurnstile(ctx context.Context, token, ip string) error {
	if !a.guestTurnstileRequired {
		return nil
	}
	if strings.TrimSpace(a.turnstileSecret) == "" {
		return errors.New("turnstile not configured")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return errTurnstileRejected
	}
	values := url.Values{}
	values.Set("secret", a.turnstileSecret)
	values.Set("response", token)
	if strings.TrimSpace(ip) != "" {
		values.Set("remoteip", ip)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.turnstileVerifyURL, bytes.NewBufferString(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New("turnstile verification unavailable")
	}
	var result turnstileSiteverifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if !result.Success {
		return errTurnstileRejected
	}
	if result.Action != "" && result.Action != guestTurnstileAction {
		return errTurnstileRejected
	}
	if a.turnstileHostname != "" && !strings.EqualFold(result.Hostname, a.turnstileHostname) {
		return errTurnstileRejected
	}
	return nil
}

var errTurnstileRejected = errors.New("turnstile rejected")
