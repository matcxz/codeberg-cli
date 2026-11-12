package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/matcxz/codeberg-cli/internal/forgejo"
	"github.com/spf13/cobra"
)

func newPRCommand() *cobra.Command {
	command := &cobra.Command{Use: "pr", Short: "Manage pull requests"}
	command.AddCommand(newPRListCommand(), newPRViewCommand(), newPRCreateCommand())
	return command
}

func newPRListCommand() *cobra.Command {
	var repository, state string
	var limit, page int
	command := &cobra.Command{
		Use:   "list",
		Short: "List pull requests in a repository",
		Args:  cobra.NoArgs,
		PreRunE: func(command *cobra.Command, _ []string) error {
			var err error
			repository, err = resolveRepository(command.Context(), repository)
			if err != nil {
				return err
			}
			return validateListOptions(repository, state, page, limit)
		},
		RunE: func(command *cobra.Command, _ []string) error {
			pulls, err := apiRead(command.Context(), func(client *forgejo.Client) ([]forgejo.PullRequest, error) {
				return client.ListPullRequests(command.Context(), repository, state, page, limit)
			})
			if err != nil {
				return err
			}
			if jsonFlag {
				return writeJSON(command.OutOrStdout(), pulls)
			}
			entries := make([][3]string, 0, len(pulls))
			for _, pull := range pulls {
				state := pull.State
				if pull.Merged {
					state = "merged"
				} else if pull.Draft {
					state = "draft"
				}
				entries = append(entries, [3]string{strconv.Itoa(pull.Index), state, pull.Title})
			}
			return printNumberedList(command.OutOrStdout(), entries)
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Repository in OWNER/NAME format (defaults to Git origin)")
	command.Flags().StringVar(&state, "state", "open", "Filter by state: open, closed, or all")
	command.Flags().IntVar(&limit, "limit", 30, "Maximum pull requests to return (1-50)")
	command.Flags().IntVar(&page, "page", 1, "Page number")
	return command
}

func newPRViewCommand() *cobra.Command {
	var repository string
	command := &cobra.Command{
		Use:   "view NUMBER",
		Short: "Show a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			repository, err := resolveRepository(command.Context(), repository)
			if err != nil {
				return err
			}
			index, err := positiveNumber(args[0])
			if err != nil {
				return err
			}
			pull, err := apiRead(command.Context(), func(client *forgejo.Client) (forgejo.PullRequest, error) {
				return client.GetPullRequest(command.Context(), repository, index)
			})
			if err != nil {
				return err
			}
			if jsonFlag {
				return writeJSON(command.OutOrStdout(), pull)
			}
			state := pull.State
			if pull.Merged {
				state = "merged"
			} else if pull.Draft {
				state = "draft"
			}
			fmt.Fprintf(command.OutOrStdout(), "#%d %s (%s)\n%s\n\n%s\n", pull.Index, pull.Title, state, pull.HTMLURL, pull.Body)
			return nil
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Repository in OWNER/NAME format (defaults to Git origin)")
	return command
}

func newPRCreateCommand() *cobra.Command {
	var repository, title, body, head, base string
	command := &cobra.Command{
		Use:   "create",
		Short: "Create a pull request",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if strings.TrimSpace(title) == "" {
				return fmt.Errorf("--title cannot be empty")
			}
			if strings.TrimSpace(head) == "" || strings.TrimSpace(base) == "" {
				return fmt.Errorf("--head and --base are required")
			}
			repository, err := resolveRepository(command.Context(), repository)
			if err != nil {
				return err
			}
			client, err := apiClient(command.Context(), true)
			if err != nil {
				return err
			}
			pull, err := client.CreatePullRequest(command.Context(), repository, forgejo.CreatePullRequestOptions{
				Head:  strings.TrimSpace(head),
				Base:  strings.TrimSpace(base),
				Title: strings.TrimSpace(title),
				Body:  body,
			})
			if err != nil {
				return err
			}
			if jsonFlag {
				return writeJSON(command.OutOrStdout(), pull)
			}
			fmt.Fprintf(command.OutOrStdout(), "Created pull request #%d: %s\n%s\n", pull.Index, pull.Title, pull.HTMLURL)
			return nil
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Repository in OWNER/NAME format (defaults to Git origin)")
	command.Flags().StringVarP(&title, "title", "t", "", "Pull request title")
	command.Flags().StringVarP(&body, "body", "b", "", "Pull request description")
	command.Flags().StringVar(&head, "head", "", "Source branch")
	command.Flags().StringVar(&base, "base", "", "Target branch")
	_ = command.MarkFlagRequired("title")
	return command
}
