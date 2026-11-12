package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/matcxz/codeberg-cli/internal/config"
	"github.com/matcxz/codeberg-cli/internal/forgejo"
	"github.com/matcxz/codeberg-cli/internal/oauth"
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
	var tokenStdin, web, noBrowser bool
	var clientID string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with a personal access token or OAuth in a browser",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			host := configuredHost(cfg)
			if web {
				if tokenStdin {
					return fmt.Errorf("--token-stdin cannot be combined with --web")
				}
				if strings.TrimSpace(os.Getenv("CODEBERG_TOKEN")) != "" {
					return fmt.Errorf("unset CODEBERG_TOKEN before using browser login")
				}
				if strings.TrimSpace(clientID) == "" {
					return fmt.Errorf("--client-id is required with --web; register a public OAuth application on this Forgejo instance")
				}
				clientID = strings.TrimSpace(clientID)
				return loginWithOAuth(cmd, cfg, host, clientID, noBrowser)
			}
			if noBrowser || clientID != "" {
				return fmt.Errorf("--no-browser and --client-id can only be used with --web")
			}

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
				cfg.OAuthClientID = ""
				cfg.RefreshToken = ""
				cfg.TokenExpiresAt = nil
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
	cmd.Flags().BoolVar(&web, "web", false, "Authenticate in a browser using OAuth authorization code with PKCE")
	cmd.Flags().StringVar(&clientID, "client-id", "", "Client ID of a public OAuth application registered on the instance")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Print the authorization URL instead of opening a browser")
	return cmd
}

func loginWithOAuth(command *cobra.Command, cfg config.Config, host, clientID string, noBrowser bool) error {
	fmt.Fprintf(command.ErrOrStderr(), "Starting secure browser authorization for %s...\n", host)
	fmt.Fprintln(command.ErrOrStderr(), "Warning: Forgejo OAuth applications do not currently support permission scopes; review the access grant before approving.")
	open := func(target string) error {
		if noBrowser {
			_, err := fmt.Fprintf(command.ErrOrStderr(), "Open this URL to authorize cb:\n%s\n", target)
			return err
		}
		return openBrowser(command, target)
	}
	token, err := oauth.Authorize(command.Context(), host, clientID, open)
	if err != nil {
		return err
	}
	client, err := forgejo.NewClient(host, token.AccessToken)
	if err != nil {
		return err
	}
	user, err := client.CurrentUser(command.Context())
	if err != nil {
		return fmt.Errorf("verify OAuth token: %w", err)
	}
	cfg.Host = host
	cfg.Username = user.Username
	cfg.Token = token.AccessToken
	cfg.OAuthClientID = clientID
	cfg.RefreshToken = token.RefreshToken
	cfg.TokenExpiresAt = nil
	if !token.ExpiresAt.IsZero() {
		cfg.TokenExpiresAt = &token.ExpiresAt
	}
	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Fprintf(command.OutOrStdout(), "Authenticated as %s on %s using OAuth.\n", user.Username, host)
	if token.RefreshToken != "" {
		fmt.Fprintln(command.OutOrStdout(), "The OAuth token will be refreshed automatically when it expires.")
	} else if !token.ExpiresAt.IsZero() {
		fmt.Fprintln(command.OutOrStdout(), "This OAuth token expires; run `cb auth login --web` again when it does.")
	}
	return nil
}

func newAuthStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the current authentication status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig(cmd.Context())
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
			cfg.OAuthClientID = ""
			cfg.RefreshToken = ""
			cfg.TokenExpiresAt = nil
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
