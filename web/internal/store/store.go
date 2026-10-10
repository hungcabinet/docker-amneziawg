package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

const (
	settingSessionSecret = "session_secret"
	settingSelectedExit  = "selected_exit"
)

// Sentinel errors for expected store conditions.
var (
	ErrNotFound         = errors.New("not found")
	ErrPasswordRequired = errors.New("PANEL_PASSWORD is required to create the first user")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrNothingToChange  = errors.New("nothing to change")
)

// User is the single panel account.
type User struct {
	Username     string
	PasswordHash string
}

// Store persists the panel account and settings in SQLite.
type Store struct {
	db *sql.DB
}

// Open creates the parent directory, opens SQLite, and applies the schema.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", filepath.ToSlash(path))
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) init() error {
	stmts := []string{
		`PRAGMA busy_timeout = 5000`,
		`PRAGMA journal_mode = WAL`,
		`PRAGMA foreign_keys = ON`,
		`CREATE TABLE IF NOT EXISTS user (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS setting (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return s.ensureSessionSecret()
}

func (s *Store) ensureSessionSecret() error {
	var value string
	err := s.db.QueryRow(`SELECT value FROM setting WHERE key = ?`, settingSessionSecret).Scan(&value)
	if err == nil && value != "" {
		return nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("read session secret: %w", err)
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Errorf("generate session secret: %w", err)
	}
	_, err = s.db.Exec(
		`INSERT INTO setting (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO NOTHING`,
		settingSessionSecret,
		hex.EncodeToString(buf),
	)
	if err != nil {
		return fmt.Errorf("save session secret: %w", err)
	}
	return nil
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}

// Bootstrap creates the first user from environment values.
// A second start leaves the existing user untouched.
func (s *Store) Bootstrap(ctx context.Context, username, password string) error {
	_, err := s.User(ctx)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrNotFound) {
		return err
	}
	if strings.TrimSpace(password) == "" {
		return ErrPasswordRequired
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("username must not be empty")
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO user (id, username, password_hash) VALUES (1, ?, ?)`,
		username,
		hash,
	)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// User returns the single account.
func (s *Store) User(ctx context.Context) (User, error) {
	var u User
	err := s.db.QueryRowContext(
		ctx,
		`SELECT username, password_hash FROM user WHERE id = 1`,
	).Scan(&u.Username, &u.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("load user: %w", err)
	}
	return u, nil
}

// Authenticate checks username and password.
func (s *Store) Authenticate(ctx context.Context, username, password string) error {
	u, err := s.User(ctx)
	if err != nil {
		return err
	}
	if u.Username != username {
		return ErrInvalidPassword
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return ErrInvalidPassword
	}
	return nil
}

// ChangeCredentials updates the username and/or password after verifying the current password.
func (s *Store) ChangeCredentials(ctx context.Context, currentPassword, newUsername, newPassword string) error {
	u, err := s.User(ctx)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(currentPassword)); err != nil {
		return ErrInvalidPassword
	}

	newUsername = strings.TrimSpace(newUsername)
	newPassword = strings.TrimSpace(newPassword)
	if newUsername == "" && newPassword == "" {
		return ErrNothingToChange
	}

	username := u.Username
	hash := u.PasswordHash
	if newUsername != "" {
		username = newUsername
	}
	if newPassword != "" {
		hash, err = hashPassword(newPassword)
		if err != nil {
			return err
		}
	}

	_, err = s.db.ExecContext(
		ctx,
		`UPDATE user SET username = ?, password_hash = ? WHERE id = 1`,
		username,
		hash,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

// SelectedExit returns the remembered exit interface name.
func (s *Store) SelectedExit(ctx context.Context) (string, error) {
	var value string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT value FROM setting WHERE key = ?`,
		settingSelectedExit,
	).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("load selected exit: %w", err)
	}
	return value, nil
}

// SetSelectedExit stores the chosen exit interface name.
func (s *Store) SetSelectedExit(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("exit name must not be empty")
	}
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO setting (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		settingSelectedExit,
		name,
	)
	if err != nil {
		return fmt.Errorf("save selected exit: %w", err)
	}
	return nil
}

// SessionSecret returns the HMAC key used for signed cookies.
func (s *Store) SessionSecret(ctx context.Context) ([]byte, error) {
	var value string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT value FROM setting WHERE key = ?`,
		settingSessionSecret,
	).Scan(&value)
	if err != nil {
		return nil, fmt.Errorf("load session secret: %w", err)
	}
	secret, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode session secret: %w", err)
	}
	return secret, nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}
