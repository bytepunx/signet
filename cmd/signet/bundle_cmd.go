package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"path"
	"strings"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	gogit "github.com/go-git/go-git/v5"
	gogitobject "github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"
)

var (
	flagBundleSecretsPath string
)

var bundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Manage local repo bundles",
}

var bundlePushCmd = &cobra.Command{
	Use:   "push [path]",
	Short: "Push a local git repo to signet as a SOPS secret bundle",
	Long: `Package a local git repository as a tar.gz archive and push it directly
to signet via the admin API. Secrets are decrypted server-side using
signet's age key, so only SOPS ciphertext ever leaves your machine.

path defaults to the current directory.

The repository must be a valid git repo with at least one commit.
Only files tracked by git (present in HEAD) are included.

Example:
  signet bundle push ./infra-secrets \
    --secrets-path secrets/ \
    --server localhost:8444 \
    --token "$TOKEN"`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath := "."
		if len(args) == 1 {
			repoPath = args[0]
		}

		ctx := cmd.Context()

		// Open the local git repository.
		repo, err := gogit.PlainOpen(repoPath)
		if err != nil {
			return fmt.Errorf("open git repo at %q: %w", repoPath, err)
		}

		head, err := repo.Head()
		if err != nil {
			return fmt.Errorf("resolve HEAD: %w (repo must have at least one commit)", err)
		}
		headSHA := head.Hash().String()

		commit, err := repo.CommitObject(head.Hash())
		if err != nil {
			return fmt.Errorf("read HEAD commit: %w", err)
		}

		tree, err := commit.Tree()
		if err != nil {
			return fmt.Errorf("read commit tree: %w", err)
		}

		secretsPrefix := strings.TrimSuffix(flagBundleSecretsPath, "/") + "/"

		// Build tar.gz in memory containing only files under secretsPrefix.
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gz)

		count := 0
		err = tree.Files().ForEach(func(f *gogitobject.File) error {
			if !strings.HasPrefix(f.Name, secretsPrefix) {
				return nil
			}
			// Skip non-YAML files — only SOPS-encrypted YAML is processed.
			if !strings.HasSuffix(f.Name, ".yaml") {
				return nil
			}

			contents, err := f.Contents()
			if err != nil {
				return fmt.Errorf("read %s: %w", f.Name, err)
			}

			hdr := &tar.Header{
				Name: path.Clean(f.Name),
				Mode: 0o600,
				Size: int64(len(contents)),
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return fmt.Errorf("tar header %s: %w", f.Name, err)
			}
			if _, err := tw.Write([]byte(contents)); err != nil {
				return fmt.Errorf("tar write %s: %w", f.Name, err)
			}
			count++
			return nil
		})
		if err != nil {
			return fmt.Errorf("build archive: %w", err)
		}

		if count == 0 {
			return fmt.Errorf("no .yaml files found under %q in HEAD commit; check --secrets-path", flagBundleSecretsPath)
		}

		if err := tw.Close(); err != nil {
			return fmt.Errorf("finalise tar: %w", err)
		}
		if err := gz.Close(); err != nil {
			return fmt.Errorf("finalise gzip: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Bundled %d file(s) from HEAD %s\n", count, headSHA[:12])

		// Stream to signet.
		conn, err := dialAdmin(ctx)
		if err != nil {
			return err
		}
		defer conn.Close()

		stream, err := gitopsClient(conn).SyncBundle(ctx)
		if err != nil {
			return fmt.Errorf("open SyncBundle stream: %w", err)
		}

		// Send header.
		if err := stream.Send(&adminv1.SyncBundleChunk{
			Payload: &adminv1.SyncBundleChunk_Header{
				Header: &adminv1.SyncBundleHeader{
					SecretsPath: flagBundleSecretsPath,
					HeadSha:     headSHA,
				},
			},
		}); err != nil {
			return fmt.Errorf("send header: %w", err)
		}

		// Stream archive in 64 KiB chunks.
		const chunkSize = 64 << 10
		data := buf.Bytes()
		for len(data) > 0 {
			n := chunkSize
			if n > len(data) {
				n = len(data)
			}
			if err := stream.Send(&adminv1.SyncBundleChunk{
				Payload: &adminv1.SyncBundleChunk_Data{Data: data[:n]},
			}); err != nil {
				return fmt.Errorf("send chunk: %w", err)
			}
			data = data[n:]
		}

		resp, err := stream.CloseAndRecv()
		if err != nil {
			return fmt.Errorf("SyncBundle: %w", err)
		}

		w := cmd.OutOrStdout()
		fmt.Fprintf(w, "Sync complete (sha: %s)\n", resp.GetSyncSha()[:12])
		fmt.Fprintf(w, "  added:   %d\n", resp.GetSecretsAdded())
		fmt.Fprintf(w, "  updated: %d\n", resp.GetSecretsUpdated())
		fmt.Fprintf(w, "  deleted: %d\n", resp.GetSecretsDeleted())
		if len(resp.GetErrors()) > 0 {
			fmt.Fprintln(w, "Errors:")
			for _, e := range resp.GetErrors() {
				fmt.Fprintf(w, "  - %s\n", e)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(bundleCmd)
	bundleCmd.AddCommand(bundlePushCmd)

	bundlePushCmd.Flags().StringVar(&flagBundleSecretsPath, "secrets-path", "secrets/",
		"directory prefix within the repo that holds SOPS-encrypted secrets")
}
