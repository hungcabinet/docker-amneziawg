package status

import (
	"context"
	"errors"
)

// ErrNotConnected is returned by the live stub until a real backend is wired.
var ErrNotConnected = errors.New("live status provider is not connected yet")

// Live is a placeholder for the real AmneziaWG backend.
type Live struct{}

// NewLive returns the live stub.
func NewLive() *Live {
	return &Live{}
}

// Server implements Provider.
func (*Live) Server(context.Context) (ServerStatus, error) {
	return ServerStatus{}, ErrNotConnected
}

// Peers implements Provider.
func (*Live) Peers(context.Context) ([]Peer, error) {
	return nil, ErrNotConnected
}

// Exits implements Provider.
func (*Live) Exits(context.Context) ([]Exit, error) {
	return nil, ErrNotConnected
}

// ExitIP implements Provider.
func (*Live) ExitIP(context.Context, string) (string, error) {
	return "", ErrNotConnected
}
