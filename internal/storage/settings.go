package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/jonsson/ccc/internal/models"
	"golang.org/x/crypto/bcrypt"
)

const (
	settingAdminPasswordHash = "admin_password_hash"
	SettingOutageThreshold   = "outage_threshold"
	SettingSiteConfig        = "site_config"
)

const (
	DefaultOutageThreshold = 0.5 // 50%
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

// GetOutageThreshold returns the outage threshold (0.0-1.0), or default if not set
func (db *DB) GetOutageThreshold() float64 {
	val, err := db.GetSetting(SettingOutageThreshold)
	if err != nil || val == "" {
		return DefaultOutageThreshold
	}
	threshold, err := strconv.ParseFloat(val, 64)
	if err != nil || threshold < 0 || threshold > 1 {
		return DefaultOutageThreshold
	}
	return threshold
}

// SetOutageThreshold sets the outage threshold (0.0-1.0)
func (db *DB) SetOutageThreshold(threshold float64) error {
	if threshold < 0 || threshold > 1 {
		return fmt.Errorf("threshold must be between 0 and 1")
	}
	return db.SetSetting(SettingOutageThreshold, strconv.FormatFloat(threshold, 'f', 2, 64))
}

// DefaultSiteConfig returns the default site configuration
func DefaultSiteConfig() models.SiteConfig {
	return models.SiteConfig{
		SiteName:        "Community Connectivity Check",
		SiteDescription: "Monitor ISP connectivity in our building",
		AboutWhy:        "",
		AboutHowItWorks: "",
		AboutPrivacy:    "",
		SupportedISPs:   []string{},
		ContactEmail:    "",
		FooterText:      "",
		GithubURL:       "",
	}
}

// GetSiteConfig returns the site configuration
func (db *DB) GetSiteConfig() (models.SiteConfig, error) {
	val, err := db.GetSetting(SettingSiteConfig)
	if err != nil {
		return DefaultSiteConfig(), err
	}
	if val == "" {
		return DefaultSiteConfig(), nil
	}

	var config models.SiteConfig
	if err := json.Unmarshal([]byte(val), &config); err != nil {
		return DefaultSiteConfig(), fmt.Errorf("failed to parse site config: %w", err)
	}
	return config, nil
}

// SetSiteConfig saves the site configuration
func (db *DB) SetSiteConfig(config models.SiteConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to serialize site config: %w", err)
	}
	return db.SetSetting(SettingSiteConfig, string(data))
}
