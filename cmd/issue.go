package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/matcxz/codeberg-cli/internal/forgejo"
	"github.com/spf13/cobra"
)

func newIssueCommand() *cobra.Command {
	command := &cobra.Command{Use: "issue", Short: "Manage repository issues"}
	command.AddCommand(newIssueListCommand(), newIssueViewCommand(), newIssueCreateCommand(),
		newIssueCommentCommand(), newIssueCloseCommand("close", "closed"), newIssueCloseCommand("reopen", "open"))
	return command
}

func newIssueListCommand() *cobra.Command {
	var repository, state string
	var limit, page int
	command := &cobra.Command{
		Use:   "list",
		Short: "List issues in a repository",
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
			issues, err := apiRead(command.Context(), func(client *forgejo.Client) ([]forgejo.Issue, error) {
				return client.ListIssues(command.Context(), repository, state, page, limit)
			})
			if err != nil {
				return err
			}
			return printIssueList(command, issues)
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Repository in OWNER/NAME format (defaults to Git origin)")
	command.Flags().StringVar(&state, "state", "open", "Filter by state: open, closed, or all")
	command.Flags().IntVar(&limit, "limit", 30, "Maximum issues to return (1-50)")
	command.Flags().IntVar(&page, "page", 1, "Page number")
	return command
}

func newIssueViewCommand() *cobra.Command {
	var repository string
	command := &cobra.Command{
		Use:   "view NUMBER",
		Short: "Show an issue",
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
			issue, err := apiRead(command.Context(), func(client *forgejo.Client) (forgejo.Issue, error) {
				return client.GetIssue(command.Context(), repository, index)
			})
			if err != nil {
				return err
			}
			if jsonFlag {
				return writeJSON(command.OutOrStdout(), issue)
			}
			fmt.Fprintf(command.OutOrStdout(), "#%d %s (%s)\n%s\n\n%s\n", issue.Index, issue.Title, issue.State, issue.HTMLURL, issue.Body)
			return nil
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Repository in OWNER/NAME format (defaults to Git origin)")
	return command
}

func newIssueCreateCommand() *cobra.Command {
	var repository, title, body string
	command := &cobra.Command{
		Use:   "create",
		Short: "Create an issue",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if strings.TrimSpace(title) == "" {
				return fmt.Errorf("--title cannot be empty")
			}
			repository, err := resolveRepository(command.Context(), repository)
			if err != nil {
				return err
			}
			client, err := apiClient(command.Context(), true)
			if err != nil {
				return err
			}
			issue, err := client.CreateIssue(command.Context(), repository, forgejo.CreateIssueOptions{
				Title: strings.TrimSpace(title),
				Body:  body,
			})
			if err != nil {
				return err
			}
			if jsonFlag {
				return writeJSON(command.OutOrStdout(), issue)
			}
			fmt.Fprintf(command.OutOrStdout(), "Created issue #%d: %s\n%s\n", issue.Index, issue.Title, issue.HTMLURL)
			return nil
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Repository in OWNER/NAME format (defaults to Git origin)")
	command.Flags().StringVarP(&title, "title", "t", "", "Issue title")
	command.Flags().StringVarP(&body, "body", "b", "", "Issue description")
	_ = command.MarkFlagRequired("title")
	return command
}

func newIssueCommentCommand() *cobra.Command {
	var repository, body string
	command := &cobra.Command{
		Use:   "comment NUMBER",
		Short: "Add a comment to an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			index, err := positiveNumber(args[0])
			if err != nil {
				return err
			}
			repository, err := resolveRepository(command.Context(), repository)
			if err != nil {
				return err
			}
			if strings.TrimSpace(body) == "" {
				return fmt.Errorf("--body cannot be empty")
			}
			client, err := apiClient(command.Context(), true)
			if err != nil {
				return err
			}
			comment, err := client.AddIssueComment(command.Context(), repository, index, body)
			if err != nil {
				return err
			}
			if jsonFlag {
				return writeJSON(command.OutOrStdout(), comment)
			}
			fmt.Fprintf(command.OutOrStdout(), "Added comment to issue #%d\n", index)
			if comment.HTMLURL != "" {
				fmt.Fprintln(command.OutOrStdout(), comment.HTMLURL)
			}
			return nil
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Repository in OWNER/NAME format (defaults to Git origin)")
	command.Flags().StringVarP(&body, "body", "b", "", "Comment text")
	_ = command.MarkFlagRequired("body")
	return command
}

func newIssueCloseCommand(use, targetState string) *cobra.Command {
	var repository string
	command := &cobra.Command{
		Use:   use + " NUMBER",
		Short: strings.ToUpper(use[:1]) + use[1:] + " an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			index, err := positiveNumber(args[0])
			if err != nil {
				return err
			}
			repository, err := resolveRepository(command.Context(), repository)
			if err != nil {
				return err
			}
			client, err := apiClient(command.Context(), true)
			if err != nil {
				return err
			}
			issue, err := client.SetIssueState(command.Context(), repository, index, targetState)
			if err != nil {
				return err
			}
			if jsonFlag {
				return writeJSON(command.OutOrStdout(), issue)
			}
			fmt.Fprintf(command.OutOrStdout(), "Issue #%d %s\n", index, targetState)
			return nil
		},
	}
	command.Flags().StringVarP(&repository, "repo", "R", "", "Repository in OWNER/NAME format (defaults to Git origin)")
	return command
}

func printIssueList(command *cobra.Command, issues []forgejo.Issue) error {
	if jsonFlag {
		return writeJSON(command.OutOrStdout(), issues)
	}
	entries := make([][3]string, 0, len(issues))
	for _, issue := range issues {
		entries = append(entries, [3]string{strconv.Itoa(issue.Index), issue.State, issue.Title})
	}
	return printNumberedList(command.OutOrStdout(), entries)
}

func validateListOptions(repository, state string, page, limit int) error {
	if _, _, ok := splitRepository(repository); !ok {
		return fmt.Errorf("--repo must be in OWNER/NAME format")
	}
	if state != "open" && state != "closed" && state != "all" {
		return fmt.Errorf("--state must be open, closed, or all")
	}
	if limit < 1 || limit > 50 {
		return fmt.Errorf("--limit must be between 1 and 50")
	}
	if page < 1 {
		return fmt.Errorf("--page must be at least 1")
	}
	return nil
}

func splitRepository(repository string) (string, string, bool) {
	owner, name, ok := strings.Cut(repository, "/")
	return owner, name, ok && owner != "" && name != "" && !strings.Contains(name, "/")
}

func positiveNumber(value string) (int, error) {
	number, err := strconv.Atoi(value)
	if err != nil || number < 1 {
		return 0, fmt.Errorf("NUMBER must be a positive integer")
	}
	return number, nil
}

func apiClient(ctx context.Context, authenticated bool) (*forgejo.Client, error) {
	cfg, err := loadConfig(ctx)
	if err != nil {
		return nil, err
	}
	if authenticated {
		if err := requireToken(cfg); err != nil {
			return nil, err
		}
	}
	return forgejo.NewClient(cfg.Host, cfg.Token)
}
