package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/matcxz/codeberg-cli/internal/forgejo"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newRepoCommand() *cobra.Command {
	repoCmd := &cobra.Command{
		Use:   "repo",
		Short: "Manage repositories",
	}
	repoCmd.AddCommand(newRepoListCommand(), newRepoViewCommand(), newRepoCreateCommand())
	return repoCmd
}

func newRepoListCommand() *cobra.Command {
	var owner string
	var limit, page int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List repositories",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig(cmd.Context())
			if err != nil {
				return err
			}
			if owner == "" {
				if err := requireToken(cfg); err != nil {
					return err
				}
			}
			repositories, err := apiRead(cmd.Context(), func(client *forgejo.Client) ([]forgejo.Repository, error) {
				return client.ListRepositories(cmd.Context(), owner, page, limit)
			})
			if err != nil {
				return err
			}
			return printRepositories(cmd.OutOrStdout(), repositories, jsonFlag)
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "List public repositories for this user")
	cmd.Flags().IntVar(&limit, "limit", 30, "Maximum repositories to return (1-50)")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.PreRunE = func(cmd *cobra.Command, _ []string) error {
		if limit < 1 || limit > 50 {
			return fmt.Errorf("--limit must be between 1 and 50")
		}
		if page < 1 {
			return fmt.Errorf("--page must be at least 1")
		}
		return nil
	}
	return cmd
}

func newRepoViewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "view OWNER/NAME",
		Short: "Show repository details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := apiRead(cmd.Context(), func(client *forgejo.Client) (forgejo.Repository, error) {
				return client.GetRepository(cmd.Context(), args[0])
			})
			if err != nil {
				return err
			}
			if jsonFlag {
				return writeJSON(cmd.OutOrStdout(), repository)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Repository:     %s\n", repository.FullName)
			fmt.Fprintf(cmd.OutOrStdout(), "Description:    %s\n", repository.Description)
			fmt.Fprintf(cmd.OutOrStdout(), "Visibility:     %s\n", visibility(repository.Private))
			fmt.Fprintf(cmd.OutOrStdout(), "Default branch: %s\n", repository.DefaultBranch)
			fmt.Fprintf(cmd.OutOrStdout(), "Stars:          %d\n", repository.Stars)
			fmt.Fprintf(cmd.OutOrStdout(), "URL:            %s\n", repository.HTMLURL)
			fmt.Fprintf(cmd.OutOrStdout(), "Clone:          %s\n", repository.CloneURL)
			return nil
		},
	}
}

func newRepoCreateCommand() *cobra.Command {
	var description string
	var private, initialize bool
	cmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a repository for the authenticated user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			if name == "" || strings.Contains(name, "/") {
				return fmt.Errorf("repository name must be a non-empty name without slashes")
			}
			cfg, err := loadConfig(cmd.Context())
			if err != nil {
				return err
			}
			if err := requireToken(cfg); err != nil {
				return err
			}
			client, err := forgejo.NewClient(cfg.Host, cfg.Token)
			if err != nil {
				return err
			}
			repository, err := client.CreateRepository(cmd.Context(), forgejo.CreateRepositoryOptions{
				Name:        name,
				Description: description,
				Private:     private,
				AutoInit:    initialize,
			})
			if err != nil {
				return err
			}
			if jsonFlag {
				return writeJSON(cmd.OutOrStdout(), repository)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n%s\n", repository.FullName, repository.HTMLURL)
			return nil
		},
	}
	cmd.Flags().StringVarP(&description, "description", "d", "", "Repository description")
	cmd.Flags().BoolVar(&private, "private", false, "Create a private repository")
	cmd.Flags().BoolVar(&initialize, "init", false, "Initialize with a README")
	return cmd
}

func printRepositories(out io.Writer, repositories []forgejo.Repository, asJSON bool) error {
	if asJSON {
		return writeJSON(out, repositories)
	}
	return printRepositoryTable(out, repositories, terminalWidth())
}

func printRepositoryTable(out io.Writer, repositories []forgejo.Repository, width int) error {
	if width < 1 {
		width = 80
	}
	writer := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	if width < 38 {
		if width < 10 {
			fmt.Fprintf(writer, "%s\n", truncate("NAME", width))
			for _, repository := range repositories {
				fmt.Fprintf(writer, "%s\n", truncate(singleLine(repository.FullName), width))
			}
			return writer.Flush()
		}
		nameWidth := width - 9
		if nameWidth < 1 {
			nameWidth = 1
		}
		fmt.Fprintf(writer, "%s\tVIS\n", truncate("NAME", nameWidth))
		for _, repository := range repositories {
			fmt.Fprintf(writer, "%s\t%s\n",
				truncate(singleLine(repository.FullName), nameWidth),
				truncate(visibility(repository.Private), 7))
		}
		return writer.Flush()
	}

	available := width - 6 - 7 - 5
	nameWidth := available * 45 / 100
	if nameWidth < 8 {
		nameWidth = 8
	}
	descriptionWidth := available - nameWidth
	if descriptionWidth < 11 {
		descriptionWidth = 11
		nameWidth = available - descriptionWidth
	}

	fmt.Fprintf(writer, "%s\tVIS\tSTARS\tDESCRIPTION\n", truncate("REPOSITORY", nameWidth))
	for _, repository := range repositories {
		fmt.Fprintf(writer, "%s\t%s\t%d\t%s\n",
			truncate(singleLine(repository.FullName), nameWidth),
			visibility(repository.Private),
			repository.Stars,
			truncate(singleLine(repository.Description), descriptionWidth))
	}
	return writer.Flush()
}

func terminalWidth() int {
	if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && width > 0 {
		return width
	}
	return 80
}

func printNumberedList(out io.Writer, entries [][3]string) error {
	return printNumberedListWidth(out, entries, terminalWidth())
}

func printNumberedListWidth(out io.Writer, entries [][3]string, width int) error {
	if width < 1 {
		width = 80
	}
	numberWidth, stateWidth := len("NUMBER"), len("STATE")
	for _, entry := range entries {
		if size := len([]rune("#" + entry[0])); size > numberWidth {
			numberWidth = size
		}
		if size := len([]rune(entry[1])); size > stateWidth {
			stateWidth = size
		}
	}
	writer := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	titleWidth := width - numberWidth - stateWidth - 4
	if titleWidth < len("TITLE") {
		fmt.Fprintln(writer, truncate("ITEMS", width))
		for _, entry := range entries {
			line := fmt.Sprintf("#%s %s %s", entry[0], entry[1], entry[2])
			fmt.Fprintln(writer, truncate(singleLine(line), width))
		}
		return writer.Flush()
	}
	fmt.Fprintf(writer, "%s\t%s\tTITLE\n", truncate("NUMBER", numberWidth), truncate("STATE", stateWidth))
	for _, entry := range entries {
		fmt.Fprintf(writer, "#%s\t%s\t%s\n", entry[0], entry[1], truncate(singleLine(entry[2]), titleWidth))
	}
	return writer.Flush()
}

func singleLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func truncate(value string, width int) string {
	if width < 1 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width <= 3 {
		return strings.Repeat(".", width)
	}
	return string(runes[:width-3]) + "..."
}

func visibility(private bool) string {
	if private {
		return "private"
	}
	return "public"
}

func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
