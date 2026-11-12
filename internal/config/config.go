package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const DefaultHost = "https://codeberg.org"

type Config struct {
	Host           string     `json:"host"`
	Token          string     `json:"token,omitempty"`
	Username       string     `json:"username,omitempty"`
	OAuthClientID  string     `json:"oauth_client_id,omitempty"`
	RefreshToken   string     `json:"refresh_token,omitempty"`
	TokenExpiresAt *time.Time `json:"token_expires_at,omitempty"`
}

func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config directory: %w", err)
	}
	return filepath.Join(dir, "cb", "config.json"), nil
}

func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}

	cfg := Config{Host: DefaultHost}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("inspect config %s: %w", path, err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0 {
		return Config{}, fmt.Errorf("config file %s is accessible by other users; run chmod 600 %s", path, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	if cfg.Host == "" {
		cfg.Host = DefaultHost
	}
	return cfg, nil
}

func Save(cfg Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if cfg.Host == "" {
		cfg.Host = DefaultHost
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	file, err := os.CreateTemp(dir, ".config-*.json")
	if err != nil {
		return fmt.Errorf("create config file: %w", err)
	}
	tmpPath := file.Name()
	defer os.Remove(tmpPath)

	if _, err := file.Write(data); err != nil {
		file.Close()
		return fmt.Errorf("write config: %w", err)
	}
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return fmt.Errorf("secure config file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close config file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}
