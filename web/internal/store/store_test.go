package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "panel.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestStore_BootstrapCreatesUserOnce(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	if err := st.Bootstrap(ctx, "admin", "first-pass"); err != nil {
		t.Fatalf("first Bootstrap() error = %v", err)
	}
	if err := st.Bootstrap(ctx, "other", "second-pass"); err != nil {
		t.Fatalf("second Bootstrap() error = %v", err)
	}

	u, err := st.User(ctx)
	if err != nil {
		t.Fatalf("User() error = %v", err)
	}
	if u.Username != "admin" {
		t.Errorf("Username = %q, want admin", u.Username)
	}
	if err := st.Authenticate(ctx, "admin", "first-pass"); err != nil {
		t.Fatalf("original password rejected: %v", err)
	}
	if err := st.Authenticate(ctx, "admin", "second-pass"); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("overwritten password accepted: %v", err)
	}
}

func TestStore_BootstrapRequiresPassword(t *testing.T) {
	st := openTestStore(t)
	err := st.Bootstrap(context.Background(), "admin", "")
	if !errors.Is(err, ErrPasswordRequired) {
		t.Fatalf("Bootstrap() error = %v, want ErrPasswordRequired", err)
	}
}

func TestStore_ChangeCredentials(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	if err := st.Bootstrap(ctx, "admin", "old-pass"); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	if err := st.ChangeCredentials(ctx, "wrong", "root", "new-pass"); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("wrong current password: error = %v", err)
	}
	if err := st.ChangeCredentials(ctx, "old-pass", "", ""); !errors.Is(err, ErrNothingToChange) {
		t.Fatalf("empty change: error = %v", err)
	}
	if err := st.ChangeCredentials(ctx, "old-pass", "root", "new-pass"); err != nil {
		t.Fatalf("ChangeCredentials() error = %v", err)
	}
	if err := st.Authenticate(ctx, "root", "new-pass"); err != nil {
		t.Fatalf("new credentials rejected: %v", err)
	}
	if err := st.Authenticate(ctx, "admin", "old-pass"); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("old credentials still work: %v", err)
	}
}

func TestStore_SelectedExitAndSessionSecret(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "panel.db")
	ctx := context.Background()

	st, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	secret1, err := st.SessionSecret(ctx)
	if err != nil {
		t.Fatalf("SessionSecret() error = %v", err)
	}
	if err := st.SetSelectedExit(ctx, "wg1"); err != nil {
		t.Fatalf("SetSelectedExit() error = %v", err)
	}
	_ = st.Close()

	st, err = Open(path)
	if err != nil {
		t.Fatalf("reopen error = %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	secret2, err := st.SessionSecret(ctx)
	if err != nil {
		t.Fatalf("SessionSecret() after reopen: %v", err)
	}
	if string(secret1) != string(secret2) {
		t.Fatal("session secret changed after reopen")
	}
	got, err := st.SelectedExit(ctx)
	if err != nil {
		t.Fatalf("SelectedExit() error = %v", err)
	}
	if got != "wg1" {
		t.Errorf("SelectedExit() = %q, want wg1", got)
	}
}
