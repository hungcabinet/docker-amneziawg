package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const sessionTTL = 24 * time.Hour

// ErrInvalidSession is returned when a cookie cannot be trusted.
var ErrInvalidSession = errors.New("invalid session")

// Session is the signed-cookie payload.
type Session struct {
	Username string    `json:"u"`
	Expiry   time.Time `json:"e"`
}

type wireSession struct {
	Username string `json:"u"`
	Expiry   int64  `json:"e"`
}

// Encode signs a session for a cookie value.
func Encode(secret []byte, sess Session) (string, error) {
	if sess.Expiry.IsZero() {
		sess.Expiry = time.Now().Add(sessionTTL)
	}
	payload, err := json.Marshal(wireSession{
		Username: sess.Username,
		Expiry:   sess.Expiry.Unix(),
	})
	if err != nil {
		return "", fmt.Errorf("encode session: %w", err)
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(payload)
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// Decode verifies a cookie value and rejects expired sessions.
func Decode(secret []byte, value string) (Session, error) {
	payloadB64, sigB64, ok := strings.Cut(value, ".")
	if !ok {
		return Session{}, ErrInvalidSession
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return Session{}, ErrInvalidSession
	}
	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return Session{}, ErrInvalidSession
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(payload)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return Session{}, ErrInvalidSession
	}
	var wire wireSession
	if err := json.Unmarshal(payload, &wire); err != nil {
		return Session{}, ErrInvalidSession
	}
	if wire.Username == "" || wire.Expiry == 0 {
		return Session{}, ErrInvalidSession
	}
	exp := time.Unix(wire.Expiry, 0)
	if time.Now().After(exp) {
		return Session{}, ErrInvalidSession
	}
	return Session{Username: wire.Username, Expiry: exp}, nil
}

// TTL is the cookie lifetime.
func TTL() time.Duration {
	return sessionTTL
}
