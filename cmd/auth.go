package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/matcxz/codeberg-cli/internal/config"
	"github.com/matcxz/codeberg-cli/internal/forgejo"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newAuthCommand() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Codeberg authentication",
	}
	authCmd.AddCommand(newAuthLoginCommand(), newAuthStatusCommand(), newAuthLogoutCommand())
	return authCmd
}

func newAuthLoginCommand() *cobra.Command {
	var tokenStdin bool
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with a personal access token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			host := configuredHost(cfg)

			token := strings.TrimSpace(os.Getenv("CODEBERG_TOKEN"))
			if token == "" && tokenStdin {
				data, err := io.ReadAll(io.LimitReader(os.Stdin, 16<<10))
				if err != nil {
					return fmt.Errorf("read token from stdin: %w", err)
				}
				token = strings.TrimSpace(string(data))
			}
			if token == "" {
				if !term.IsTerminal(int(os.Stdin.Fd())) {
					return fmt.Errorf("no token provided; set CODEBERG_TOKEN or use --token-stdin")
				}
				fmt.Fprint(cmd.ErrOrStderr(), "Personal access token: ")
				data, err := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Fprintln(cmd.ErrOrStderr())
				if err != nil {
					return fmt.Errorf("read token: %w", err)
				}
				token = strings.TrimSpace(string(data))
			}
			if token == "" {
				return fmt.Errorf("token cannot be empty")
			}

			client, err := forgejo.NewClient(host, token)
			if err != nil {
				return err
			}
			user, err := client.CurrentUser(cmd.Context())
			if err != nil {
				return fmt.Errorf("authenticate: %w", err)
			}
			cfg.Host = host
			cfg.Username = user.Username
			if os.Getenv("CODEBERG_TOKEN") == "" {
				cfg.Token = token
				if err := config.Save(cfg); err != nil {
					return err
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Authenticated as %s on %s\n", user.Username, host)
			if os.Getenv("CODEBERG_TOKEN") != "" {
				fmt.Fprintln(cmd.OutOrStdout(), "Token supplied by CODEBERG_TOKEN; it was not saved.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&tokenStdin, "token-stdin", false, "Read the token from standard input")
	return cmd
}

func newAuthStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the current authentication status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			if cfg.Token == "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Not authenticated on %s\n", cfg.Host)
				return nil
			}
			client, err := forgejo.NewClient(cfg.Host, cfg.Token)
			if err != nil {
				return err
			}
			user, err := client.CurrentUser(cmd.Context())
			if err != nil {
				return fmt.Errorf("check authentication: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Authenticated as %s on %s\n", user.Username, cfg.Host)
			return nil
		},
	}
}

func newAuthLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove the saved Codeberg token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.Token = ""
			cfg.Username = ""
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Saved credentials removed.")
			if os.Getenv("CODEBERG_TOKEN") != "" {
				fmt.Fprintln(cmd.OutOrStdout(), "CODEBERG_TOKEN is still set and will continue to authenticate requests.")
			}
			return nil
		},
	}
}

func configuredHost(cfg config.Config) string {
	if hostFlag != "" {
		return hostFlag
	}
	if host := strings.TrimSpace(os.Getenv("CODEBERG_HOST")); host != "" {
		return host
	}
	return cfg.Host
}
