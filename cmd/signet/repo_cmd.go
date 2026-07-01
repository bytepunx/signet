package main

import (
	"context"
	"fmt"
	"os"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/spf13/cobra"
)

func init() {
	repoCmd.AddCommand(repoAddCmd, repoListCmd, repoRemoveCmd, repoSyncCmd)
	rootCmd.AddCommand(repoCmd)
}

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage git repository registrations",
}

var repoAddFlags struct {
	name         string
	repoURL      string
	branch       string
	secretsPath  string
	configPath   string
	deployKey    string
	setupWebhook bool
	githubToken  string
}

var repoAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Register a new git repository",
	Long: `Register a repository with signet and receive a GitHub webhook URL and secret.

The webhook secret is shown once and cannot be retrieved again. Store it
securely before the command exits.

Automatic webhook creation (--setup-webhook):

  When --setup-webhook is set, signet will attempt to create the webhook in
  GitHub on your behalf rather than printing manual instructions. It tries
  each credential source in order and logs every step:

    1. gh CLI  — if gh is installed and authenticated, 'gh api' is used.
       No extra credentials are needed.

    2. --github-token flag  — if gh is unavailable or fails, a GitHub personal
       access token supplied via --github-token is used directly against the
       GitHub REST API.

    3. GITHUB_TOKEN / GH_TOKEN env vars  — checked when neither of the above
       is available.

  The token (or gh identity) must have the 'admin:repo_hook' scope (classic PAT)
  or 'Repository: Administration → write' permission (fine-grained PAT).

  If none of the above are available, signet falls back to printing step-by-step
  manual instructions — it never fails silently.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if repoAddFlags.name == "" {
			return fmt.Errorf("--name is required")
		}
		if repoAddFlags.repoURL == "" {
			return fmt.Errorf("--repo-url is required")
		}

		// Read deploy key from path or stdin ("-").
		var deployKeyPEM []byte
		if repoAddFlags.deployKey == "" {
			return fmt.Errorf("--deploy-key is required")
		}
		var err error
		if repoAddFlags.deployKey == "-" {
			deployKeyPEM, err = os.ReadFile("/dev/stdin")
		} else {
			deployKeyPEM, err = os.ReadFile(repoAddFlags.deployKey)
		}
		if err != nil {
			return fmt.Errorf("read deploy key: %w", err)
		}

		ctx := context.Background()
		conn, err := dialAdmin()
		if err != nil {
			return err
		}
		defer conn.Close()

		resp, err := gitopsClient(conn).RegisterRepository(ctx, &adminv1.RegisterRepositoryRequest{
			Name:        repoAddFlags.name,
			RepoUrl:     repoAddFlags.repoURL,
			Branch:      repoAddFlags.branch,
			SecretsPath: repoAddFlags.secretsPath,
			ConfigPath:  repoAddFlags.configPath,
			DeployKey:   deployKeyPEM,
		})
		if err != nil {
			return err
		}

		w := cmd.OutOrStdout()
		fmt.Fprintf(w, "Repository registered.\n")
		fmt.Fprintf(w, "  ID:             %s\n", resp.GetId())
		fmt.Fprintf(w, "  Webhook URL:    %s\n", resp.GetWebhookUrl())
		fmt.Fprintf(w, "  Webhook secret: %s\n", resp.GetWebhookSecret())
		fmt.Fprintln(w, "\nThis secret will not be shown again.")

		if repoAddFlags.setupWebhook {
			owner, repo, err := parseGitHubRepo(repoAddFlags.repoURL)
			if err != nil {
				fmt.Fprintf(w, "\n--setup-webhook: %v\n", err)
				printManualWebhookInstructions(w, resp.GetWebhookUrl())
			} else {
				tryCreateGitHubWebhook(ctx, w, resp.GetWebhookUrl(), resp.GetWebhookSecret(), owner, repo, repoAddFlags.githubToken)
			}
		} else {
			fmt.Fprintln(w, "\nConfigure the webhook in GitHub → Settings → Webhooks.")
			fmt.Fprintln(w, "Run with --setup-webhook to create it automatically.")
		}
		return nil
	},
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered repositories",
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := context.Background()
		conn, err := dialAdmin()
		if err != nil {
			return err
		}
		defer conn.Close()

		resp, err := gitopsClient(conn).ListRepositories(ctx, &adminv1.ListRepositoriesRequest{})
		if err != nil {
			return err
		}
		if len(resp.GetRepositories()) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No repositories registered. Run 'signet repo add'.")
			return nil
		}
		for _, r := range resp.GetRepositories() {
			lastSync := "-"
			if t := r.GetLastSyncAt(); t != "" {
				lastSync = t
			}
			sha := r.GetLastSyncSha()
			if len(sha) > 8 {
				sha = sha[:8]
			}
			configInfo := ""
			if cp := r.GetConfigPath(); cp != "" {
				configInfo = "  config=" + cp
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s  %s  [%s@%s]  last-sync=%s sha=%s%s\n",
				r.GetId(), r.GetName(), r.GetBranch(), r.GetSecretsPath(), lastSync, sha, configInfo)
		}
		return nil
	},
}

var repoRemoveFlags struct {
	id string
}

var repoRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a repository registration",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if repoRemoveFlags.id == "" {
			return fmt.Errorf("--id is required")
		}
		ctx := context.Background()
		conn, err := dialAdmin()
		if err != nil {
			return err
		}
		defer conn.Close()

		resp, err := gitopsClient(conn).RemoveRepository(ctx, &adminv1.RemoveRepositoryRequest{
			Id: repoRemoveFlags.id,
		})
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), resp.GetMessage())
		return nil
	},
}

var repoSyncFlags struct {
	id string
}

var repoSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Trigger an immediate full sync of a repository",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if repoSyncFlags.id == "" {
			return fmt.Errorf("--id is required")
		}
		ctx := context.Background()
		conn, err := dialAdmin()
		if err != nil {
			return err
		}
		defer conn.Close()

		resp, err := gitopsClient(conn).TriggerSync(ctx, &adminv1.TriggerSyncRequest{
			Id: repoSyncFlags.id,
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Sync complete.\n  SHA:     %s\n  Added:   %d\n  Updated: %d\n  Deleted: %d\n",
			resp.GetSyncSha(), resp.GetSecretsAdded(), resp.GetSecretsUpdated(), resp.GetSecretsDeleted())
		return nil
	},
}

func init() {
	repoAddCmd.Flags().StringVar(&repoAddFlags.name, "name", "", "human alias for the repository (required)")
	repoAddCmd.Flags().StringVar(&repoAddFlags.repoURL, "repo-url", "", "repository URL, e.g. git@github.com:org/repo (required)")
	repoAddCmd.Flags().StringVar(&repoAddFlags.branch, "branch", "main", "branch to track")
	repoAddCmd.Flags().StringVar(&repoAddFlags.secretsPath, "secrets-path", "secrets/", "path within repo where SOPS-encrypted secrets live")
	repoAddCmd.Flags().StringVar(&repoAddFlags.configPath, "config-path", "", "path within repo where plain YAML config files live, e.g. \"config/\" (optional)")
	repoAddCmd.Flags().StringVar(&repoAddFlags.deployKey, "deploy-key", "", "path to PEM-encoded SSH deploy key, or \"-\" for stdin (required)")
	repoAddCmd.Flags().BoolVar(&repoAddFlags.setupWebhook, "setup-webhook", false,
		"automatically create the GitHub webhook after registration (requires gh CLI or a GitHub token)")
	repoAddCmd.Flags().StringVar(&repoAddFlags.githubToken, "github-token", "",
		"GitHub personal access token for webhook creation (admin:repo_hook scope); falls back to GITHUB_TOKEN / GH_TOKEN env vars")

	repoRemoveCmd.Flags().StringVar(&repoRemoveFlags.id, "id", "", "repository ID (required)")
	repoSyncCmd.Flags().StringVar(&repoSyncFlags.id, "id", "", "repository ID (required)")
}
