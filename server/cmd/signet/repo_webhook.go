package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// parseGitHubRepo extracts the owner and repository name from a GitHub URL.
// Supports SSH (git@github.com:owner/repo[.git]) and HTTPS formats.
// Returns a non-nil error for non-GitHub hosts so the caller can skip webhook
// creation and print manual instructions instead.
func parseGitHubRepo(rawURL string) (owner, repo string, err error) {
	// SSH format: git@<host>:<owner>/<repo>[.git]
	if strings.HasPrefix(rawURL, "git@") {
		at := strings.Index(rawURL, "@")
		colon := strings.Index(rawURL, ":")
		if colon <= at {
			return "", "", fmt.Errorf("cannot parse SSH URL %q", rawURL)
		}
		host := rawURL[at+1 : colon]
		if host != "github.com" {
			return "", "", fmt.Errorf("--repo-url host is %q, not github.com; automatic webhook creation is only supported for GitHub repositories", host)
		}
		path := strings.TrimSuffix(rawURL[colon+1:], ".git")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("cannot parse owner/repo from SSH URL %q", rawURL)
		}
		return parts[0], parts[1], nil
	}
	u, parseErr := url.Parse(rawURL)
	if parseErr != nil {
		return "", "", fmt.Errorf("parse URL %q: %w", rawURL, parseErr)
	}
	if u.Host != "github.com" {
		return "", "", fmt.Errorf("--repo-url host is %q, not github.com; automatic webhook creation is only supported for GitHub repositories", u.Host)
	}
	p := strings.Trim(u.Path, "/")
	p = strings.TrimSuffix(p, ".git")
	parts := strings.SplitN(p, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("cannot parse owner/repo from HTTPS URL %q", rawURL)
	}
	return parts[0], parts[1], nil
}

// resolveGitHubToken returns the first available GitHub API token and a
// human-readable description of its source.
// Priority: --github-token flag → GITHUB_TOKEN env → GH_TOKEN env.
func resolveGitHubToken(flagToken string) (token, description string) {
	if flagToken != "" {
		return flagToken, "--github-token flag was supplied"
	}
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		return t, "GITHUB_TOKEN is set in environment"
	}
	if t := os.Getenv("GH_TOKEN"); t != "" {
		return t, "GH_TOKEN is set in environment"
	}
	return "", ""
}

// tryCreateGitHubWebhook attempts to register a GitHub push webhook for the
// given repository. It logs each phase to w so the operator can see exactly
// what is being attempted and why. It never returns an error — failures fall
// back to printing manual setup instructions.
//
// Phase 1: gh CLI (exec.LookPath → gh api)
// Phase 2: direct GitHub REST API (--github-token / GITHUB_TOKEN / GH_TOKEN)
// Fallback: print manual instructions
func tryCreateGitHubWebhook(ctx context.Context, w io.Writer, webhookURL, secret, owner, repo, githubToken string) {
	type hookConfig struct {
		URL         string `json:"url"`
		ContentType string `json:"content_type"`
		Secret      string `json:"secret"`
		InsecureSSL string `json:"insecure_ssl"`
	}
	type hookPayload struct {
		Name   string     `json:"name"`
		Config hookConfig `json:"config"`
		Events []string   `json:"events"`
		Active bool       `json:"active"`
	}
	payload, _ := json.Marshal(hookPayload{
		Name: "web",
		Config: hookConfig{
			URL:         webhookURL,
			ContentType: "json",
			Secret:      secret,
			InsecureSSL: "0",
		},
		Events: []string{"push"},
		Active: true,
	})
	apiPath := fmt.Sprintf("repos/%s/%s/hooks", owner, repo)

	// ── Phase 1: gh CLI ──────────────────────────────────────────────────────
	fmt.Fprintf(w, "\nSetting up webhook automatically for %s/%s...\n", owner, repo)
	fmt.Fprintf(w, "  Checking for gh CLI...")

	ghPath, lookErr := exec.LookPath("gh")
	ghFound := lookErr == nil

	if !ghFound {
		fmt.Fprintln(w, " not found.")
	} else {
		fmt.Fprintf(w, " found (%s).\n", ghPath)
		fmt.Fprintf(w, "  Calling `gh api %s`...\n", apiPath)

		cmd := exec.CommandContext(ctx, "gh", "api", apiPath, "--method", "POST", "--input", "-")
		cmd.Stdin = bytes.NewReader(payload)
		out, err := cmd.CombinedOutput()
		if err == nil {
			fmt.Fprintln(w, "  Webhook created.")
			return
		}
		fmt.Fprintf(w, "  gh api failed: %s\n", strings.TrimSpace(string(out)))
		fmt.Fprintln(w, "  Falling back to GitHub REST API...")
	}

	// ── Phase 2: direct REST API ─────────────────────────────────────────────
	token, tokenDesc := resolveGitHubToken(githubToken)
	if token == "" {
		if !ghFound {
			fmt.Fprintln(w, "  gh CLI not available and no GitHub token found.")
			fmt.Fprintln(w, "  Set --github-token, GITHUB_TOKEN, or GH_TOKEN, or install and authenticate gh.")
		} else {
			fmt.Fprintln(w, "  No GitHub token available for REST API fallback.")
			fmt.Fprintln(w, "  Set --github-token, GITHUB_TOKEN, or GH_TOKEN.")
		}
		printManualWebhookInstructions(w, webhookURL)
		return
	}

	if !ghFound {
		fmt.Fprintf(w, "  gh CLI not available, %s, calling GitHub's REST API...\n", tokenDesc)
	} else {
		fmt.Fprintf(w, "  %s, calling GitHub's REST API...\n", tokenDesc)
	}

	// GITHUB_API_BASE can be set in tests to redirect to a local HTTP server.
	apiBase := os.Getenv("GITHUB_API_BASE")
	if apiBase == "" {
		apiBase = "https://api.github.com"
	}
	apiURL := fmt.Sprintf("%s/%s", strings.TrimRight(apiBase, "/"), apiPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		fmt.Fprintf(w, "  Failed to build API request: %v\n", err)
		printManualWebhookInstructions(w, webhookURL)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(w, "  GitHub REST API request failed: %v\n", err)
		printManualWebhookInstructions(w, webhookURL)
		return
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated:
		fmt.Fprintln(w, "  Webhook created.")
	case http.StatusUnprocessableEntity:
		fmt.Fprintln(w, "  Webhook already exists for this URL (GitHub returned 422 Unprocessable Entity).")
	default:
		var body bytes.Buffer
		_, _ = body.ReadFrom(io.LimitReader(resp.Body, 512))
		fmt.Fprintf(w, "  GitHub API returned HTTP %d: %s\n", resp.StatusCode, strings.TrimSpace(body.String()))
		printManualWebhookInstructions(w, webhookURL)
	}
}

// printManualWebhookInstructions prints step-by-step instructions for manually
// configuring the webhook in GitHub.
func printManualWebhookInstructions(w io.Writer, webhookURL string) {
	fmt.Fprintln(w, "\n  To configure the webhook manually:")
	fmt.Fprintln(w, "    1. Open your repository on GitHub")
	fmt.Fprintln(w, "    2. Go to Settings → Webhooks → Add webhook")
	fmt.Fprintf(w, "    3. Payload URL:   %s\n", webhookURL)
	fmt.Fprintln(w, "    4. Content type: application/json")
	fmt.Fprintln(w, "    5. Secret:       (the webhook secret printed above)")
	fmt.Fprintln(w, "    6. Events:       Just the push event")
	fmt.Fprintln(w, "    7. Active:       checked → Add webhook")
}
