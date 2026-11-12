package cmd

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/matcxz/codeberg-cli/internal/config"
	"github.com/spf13/cobra"
)

func newBrowseCommand() *cobra.Command {
	var noBrowser bool
	command := &cobra.Command{
		Use:   "browse OWNER/NAME [LOCATION]",
		Short: "Open a repository, issue, or pull request in a browser",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(command *cobra.Command, args []string) error {
			owner, name, ok := splitRepository(args[0])
			if !ok {
				return fmt.Errorf("repository must be in OWNER/NAME format")
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.Host = configuredHost(cfg)
			parts := []string{strings.TrimRight(cfg.Host, "/"), owner, name}
			if len(args) == 2 {
				location := strings.Trim(args[1], "/")
				if location == "" || strings.HasPrefix(location, ".") || strings.Contains(location, "..") ||
					strings.ContainsAny(location, "?#") {
					return fmt.Errorf("LOCATION must be a repository-relative path")
				}
				parts = append(parts, location)
			}
			target, err := url.JoinPath(parts[0], parts[1:]...)
			if err != nil {
				return fmt.Errorf("build repository URL: %w", err)
			}
			if noBrowser {
				_, err := fmt.Fprintln(command.OutOrStdout(), target)
				return err
			}
			return openBrowser(command, target)
		},
	}
	command.Flags().BoolVar(&noBrowser, "no-browser", false, "Print the URL instead of opening a browser")
	return command
}

func openBrowser(command *cobra.Command, target string) error {
	var executable string
	var args []string
	if browser := strings.TrimSpace(os.Getenv("BROWSER")); browser != "" {
		fields := strings.Fields(browser)
		executable, args = fields[0], fields[1:]
	} else {
		switch runtime.GOOS {
		case "darwin":
			executable = "open"
		case "windows":
			executable = "rundll32"
			args = []string{"url.dll,FileProtocolHandler"}
		default:
			executable = "xdg-open"
		}
	}
	args = append(args, target)
	process := exec.CommandContext(command.Context(), executable, args...)
	if err := process.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	if err := process.Process.Release(); err != nil {
		return fmt.Errorf("release browser process: %w", err)
	}
	return nil
}
