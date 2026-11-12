package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/matcxz/codeberg-cli/internal/config"
	"github.com/matcxz/codeberg-cli/internal/forgejo"
	"github.com/matcxz/codeberg-cli/internal/oauth"
	"github.com/spf13/cobra"
)

var (
	hostFlag string
	jsonFlag bool
)

var rootCmd = &cobra.Command{
	Use:           "cb",
	Short:         "A command-line client for Codeberg",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Version = "0.1.0"
	rootCmd.PersistentFlags().StringVar(&hostFlag, "host", "", "Codeberg instance URL")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output JSON")
	rootCmd.AddCommand(newAuthCommand(), newRepoCommand(), newIssueCommand(), newPRCommand(), newBrowseCommand())
}

func loadConfig(ctx context.Context) (config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return cfg, err
	}
	if token := strings.TrimSpace(os.Getenv("CODEBERG_TOKEN")); token != "" {
		cfg.Token = token
		cfg.OAuthClientID = ""
		cfg.RefreshToken = ""
		cfg.TokenExpiresAt = nil
	} else {
		selectedHost := configuredHost(cfg)
		if selectedHost != cfg.Host {
			cfg.Token = ""
			cfg.OAuthClientID = ""
			cfg.RefreshToken = ""
			cfg.TokenExpiresAt = nil
		} else if cfg.TokenExpiresAt != nil && time.Until(*cfg.TokenExpiresAt) < time.Minute {
			if cfg.RefreshToken == "" || cfg.OAuthClientID == "" {
				return config.Config{}, fmt.Errorf("saved OAuth token expired; run `cb auth login --web` again")
			}
			refreshed, err := oauth.Refresh(ctx, cfg.Host, cfg.OAuthClientID, cfg.RefreshToken)
			if err != nil {
				return config.Config{}, fmt.Errorf("refresh OAuth token; run `cb auth login --web` again: %w", err)
			}
			cfg.Token = refreshed.AccessToken
			cfg.RefreshToken = refreshed.RefreshToken
			cfg.TokenExpiresAt = nil
			if !refreshed.ExpiresAt.IsZero() {
				cfg.TokenExpiresAt = &refreshed.ExpiresAt
			}
			if err := config.Save(cfg); err != nil {
				return config.Config{}, fmt.Errorf("save refreshed OAuth token: %w", err)
			}
		}
	}
	cfg.Host = configuredHost(cfg)
	return cfg, nil
}

func requireToken(cfg config.Config) error {
	if strings.TrimSpace(cfg.Token) == "" {
		return fmt.Errorf("not authenticated; run `cb auth login` or set CODEBERG_TOKEN")
	}
	return nil
}

func apiRead[T any](ctx context.Context, operation func(*forgejo.Client) (T, error)) (T, error) {
	cfg, err := loadConfig(ctx)
	if err != nil {
		var zero T
		return zero, err
	}
	client, err := forgejo.NewClient(cfg.Host, cfg.Token)
	if err != nil {
		var zero T
		return zero, err
	}
	result, err := operation(client)
	if !forgejo.IsInsufficientScope(err) {
		return result, err
	}

	publicClient, err := forgejo.NewClient(cfg.Host, "")
	if err != nil {
		var zero T
		return zero, err
	}
	result, retryErr := operation(publicClient)
	if retryErr != nil {
		var zero T
		return zero, fmt.Errorf("%w (unauthenticated retry also failed: %v)", err, retryErr)
	}
	return result, nil
}
