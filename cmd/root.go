package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/matcxz/codeberg-cli/internal/config"
	"github.com/matcxz/codeberg-cli/internal/forgejo"
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

func loadConfig() (config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return cfg, err
	}
	cfg.Host = configuredHost(cfg)
	if token := strings.TrimSpace(os.Getenv("CODEBERG_TOKEN")); token != "" {
		cfg.Token = token
	}
	return cfg, nil
}

func requireToken(cfg config.Config) error {
	if strings.TrimSpace(cfg.Token) == "" {
		return fmt.Errorf("not authenticated; run `cb auth login` or set CODEBERG_TOKEN")
	}
	return nil
}

func apiRead[T any](operation func(*forgejo.Client) (T, error)) (T, error) {
	cfg, err := loadConfig()
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
