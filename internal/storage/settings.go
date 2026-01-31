package storage

import (
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	settingAdminPasswordHash = "admin_password_hash"
)

// SetAdminPassword sets the admin password (stores bcrypt hash)
func (db *DB) SetAdminPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	_, err = db.conn.Exec(`
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, settingAdminPasswordHash, string(hash))
	if err != nil {
		return fmt.Errorf("failed to save password: %w", err)
	}

	return nil
}

// CheckAdminPassword verifies the admin password
// Returns true if password matches, false otherwise
// Returns error only on database errors
func (db *DB) CheckAdminPassword(password string) (bool, error) {
	var hash string
	err := db.conn.QueryRow(`SELECT value FROM settings WHERE key = ?`, settingAdminPasswordHash).Scan(&hash)
	if err == sql.ErrNoRows {
		return false, nil // No password set
	}
	if err != nil {
		return false, fmt.Errorf("failed to get password: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil, nil
}

// HasAdminPassword checks if an admin password has been set
func (db *DB) HasAdminPassword() (bool, error) {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM settings WHERE key = ?`, settingAdminPasswordHash).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check password: %w", err)
	}
	return count > 0, nil
}

// GetSetting gets a setting value by key
func (db *DB) GetSetting(key string) (string, error) {
	var value string
	err := db.conn.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get setting: %w", err)
	}
	return value, nil
}

// SetSetting sets a setting value
func (db *DB) SetSetting(key, value string) error {
	_, err := db.conn.Exec(`
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value)
	if err != nil {
		return fmt.Errorf("failed to save setting: %w", err)
	}
	return nil
}
