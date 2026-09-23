package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

// lookPath resolves the sops binary on PATH. Overridable in tests.
var lookPath = exec.LookPath

// runSops invokes the sops binary with dir as its working directory,
// returning combined stdout+stderr. Overridable in tests.
var runSops = func(dir string, args []string) ([]byte, error) {
	sopsPath, err := lookPath("sops")
	if err != nil {
		return nil, fmt.Errorf("sops binary not found on PATH: install it from https://github.com/getsops/sops#download")
	}
	cmd := exec.Command(sopsPath, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

var secretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Author SOPS-encrypted secret files in a local repository checkout",
	Long: `Create or remove SOPS-encrypted secret files at the path convention signet
expects (<secrets-root>/[<env>/]<namespace>/<service>/<name>.yaml), without needing
to hand-write SOPS metadata or learn its CLI directly.

These commands operate on local files only — they do not talk to a signet server.
Run them from anywhere inside a checkout that already has a .sops.yaml (created by
'signet sops-key update-config'); the repository root and environment are detected
automatically. Encryption is performed by shelling out to the sops binary, which
must be installed and on PATH.`,
}

var secretSetFlags struct {
	value       string
	valueFile   string
	env         string
	secretsRoot string
	sopsConfig  string
}

var secretRmFlags struct {
	env         string
	secretsRoot string
	sopsConfig  string
}

func init() {
	secretSetCmd.Flags().StringVar(&secretSetFlags.value, "value", "", "secret value (avoid on shared/logged shells; prefer --value-file or stdin)")
	secretSetCmd.Flags().StringVar(&secretSetFlags.valueFile, "value-file", "", "path to a file containing the secret value")
	secretSetCmd.Flags().StringVar(&secretSetFlags.env, "env", "", "environment to write under (auto-detected from .sops.yaml if omitted)")
	secretSetCmd.Flags().StringVar(&secretSetFlags.secretsRoot, "secrets-root", "secrets/", "secrets directory prefix within the repository")
	secretSetCmd.Flags().StringVar(&secretSetFlags.sopsConfig, "sops-config", "", "path to .sops.yaml (auto-discovered by walking up from the current directory if omitted)")

	secretRmCmd.Flags().StringVar(&secretRmFlags.env, "env", "", "environment the secret was written under (auto-detected from .sops.yaml if omitted)")
	secretRmCmd.Flags().StringVar(&secretRmFlags.secretsRoot, "secrets-root", "secrets/", "secrets directory prefix within the repository")
	secretRmCmd.Flags().StringVar(&secretRmFlags.sopsConfig, "sops-config", "", "path to .sops.yaml (auto-discovered by walking up from the current directory if omitted)")

	secretCmd.AddCommand(secretSetCmd, secretRmCmd)
	rootCmd.AddCommand(secretCmd)
}

var secretSetCmd = &cobra.Command{
	Use:   "set <namespace>/<service>/<name>",
	Short: "Create or update a SOPS-encrypted secret file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSecretSet(cmd, cmd.InOrStdin(), args[0])
	},
}

var secretRmCmd = &cobra.Command{
	Use:   "rm <namespace>/<service>/<name>",
	Short: "Delete a secret file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSecretRm(cmd, args[0])
	},
}

// splitSecretRef parses "<namespace>/<service>/<name>" into its components.
func splitSecretRef(ref string) (namespace, service, name string, err error) {
	parts := strings.Split(ref, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", fmt.Errorf("expected <namespace>/<service>/<name>, got %q", ref)
	}
	return parts[0], parts[1], parts[2], nil
}

// findSOPSConfig walks upward from startDir looking for a .sops.yaml file,
// returning the directory that contains it (the repository root) and the
// full path to the file.
func findSOPSConfig(startDir string) (repoRoot, configPath string, err error) {
	dir, absErr := filepath.Abs(startDir)
	if absErr != nil {
		return "", "", absErr
	}
	for {
		candidate := filepath.Join(dir, ".sops.yaml")
		if _, statErr := os.Stat(candidate); statErr == nil {
			return dir, candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", fmt.Errorf("no .sops.yaml found in %q or any parent directory; run inside a signet secrets repository, or run 'signet sops-key update-config' first", startDir)
		}
		dir = parent
	}
}

// resolveRepoRoot determines the repository root and .sops.yaml path. If
// sopsConfigFlag is set it is used directly (repoRoot is its directory);
// otherwise .sops.yaml is discovered by walking up from cwd.
func resolveRepoRoot(sopsConfigFlag, cwd string) (repoRoot, configPath string, err error) {
	if sopsConfigFlag == "" {
		return findSOPSConfig(cwd)
	}
	abs, err := filepath.Abs(sopsConfigFlag)
	if err != nil {
		return "", "", err
	}
	return filepath.Dir(abs), abs, nil
}

// resolveEnvironment determines which environment a secret should be written
// under. An explicit envFlag always wins. Otherwise the environments map in
// the .sops.yaml at configPath is consulted: empty means a single-environment
// repository (no env segment), exactly one entry is auto-selected, and more
// than one requires the caller to disambiguate via --env.
func resolveEnvironment(configPath, envFlag string) (env string, autoSelected bool, err error) {
	if envFlag != "" {
		return envFlag, false, nil
	}

	doc, err := loadSOPSDoc(configPath)
	if err != nil {
		return "", false, fmt.Errorf("load %s: %w", configPath, err)
	}
	root := docRoot(doc)
	_, envMap := mappingGet(root, "environments")
	if envMap == nil || len(envMap.Content) == 0 {
		return "", false, nil
	}

	var names []string
	for i := 0; i+1 < len(envMap.Content); i += 2 {
		names = append(names, envMap.Content[i].Value)
	}
	if len(names) == 1 {
		return names[0], true, nil
	}
	return "", false, fmt.Errorf("multiple environments found in %s (%s); specify one with --env", configPath, strings.Join(names, ", "))
}

// resolveSecretPath builds the repository-root-relative path for a secret,
// following the <secrets-root>/[<env>/]<namespace>/<service>/<name>.yaml
// convention documented in internal/gitops/path.go.
func resolveSecretPath(secretsRoot, env, namespace, service, name string) string {
	return filepath.ToSlash(filepath.Join(secretsRoot, env, namespace, service, name+".yaml"))
}

// resolveSecretValue determines the plaintext secret value from, in order:
// the --value flag, the --value-file flag, piped stdin, or an interactive
// masked prompt.
func resolveSecretValue(cmd *cobra.Command, stdin io.Reader, value, valueFile string) (string, error) {
	if value != "" && valueFile != "" {
		return "", fmt.Errorf("only one of --value or --value-file may be given")
	}
	if value != "" {
		return value, nil
	}
	if valueFile != "" {
		data, err := os.ReadFile(valueFile)
		if err != nil {
			return "", fmt.Errorf("read --value-file %s: %w", valueFile, err)
		}
		return strings.TrimSuffix(strings.TrimSuffix(string(data), "\n"), "\r"), nil
	}

	if f, ok := stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		fmt.Fprint(cmd.OutOrStdout(), "Secret value: ")
		data, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(cmd.OutOrStdout())
		if err != nil {
			return "", fmt.Errorf("read secret value: %w", err)
		}
		return string(data), nil
	}

	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	return strings.TrimSuffix(strings.TrimSuffix(string(data), "\n"), "\r"), nil
}

func runSecretSet(cmd *cobra.Command, stdin io.Reader, ref string) error {
	namespace, service, name, err := splitSecretRef(ref)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	repoRoot, configPath, err := resolveRepoRoot(secretSetFlags.sopsConfig, cwd)
	if err != nil {
		return err
	}

	env, autoSelected, err := resolveEnvironment(configPath, secretSetFlags.env)
	if err != nil {
		return err
	}

	relPath := resolveSecretPath(secretSetFlags.secretsRoot, env, namespace, service, name)
	absPath := filepath.Join(repoRoot, filepath.FromSlash(relPath))

	value, err := resolveSecretValue(cmd, stdin, secretSetFlags.value, secretSetFlags.valueFile)
	if err != nil {
		return err
	}
	if value == "" {
		return fmt.Errorf("secret value must not be empty")
	}

	plain, err := yaml.Marshal(map[string]string{"value": value})
	if err != nil {
		return fmt.Errorf("marshal plaintext yaml: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", relPath, err)
	}
	if err := os.WriteFile(absPath, plain, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", relPath, err)
	}

	out, sopsErr := runSops(repoRoot, []string{"--config", configPath, "--encrypt", "--in-place", relPath})
	if sopsErr != nil {
		_ = os.Remove(absPath)
		if len(out) > 0 {
			return fmt.Errorf("sops encrypt failed: %s", strings.TrimSpace(string(out)))
		}
		return fmt.Errorf("sops encrypt failed: %w", sopsErr)
	}

	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "Wrote encrypted secret: %s\n", relPath)
	envDisplay := env
	if envDisplay == "" {
		envDisplay = "(global)"
	}
	if autoSelected {
		fmt.Fprintf(w, "  environment: %s (auto-detected)\n", envDisplay)
	} else {
		fmt.Fprintf(w, "  environment: %s\n", envDisplay)
	}
	fmt.Fprintf(w, "\nNext: git add %s && git commit && git push\n", relPath)
	fmt.Fprintln(w, "(or 'signet bundle push' if this repository has no remote yet)")
	return nil
}

func runSecretRm(cmd *cobra.Command, ref string) error {
	namespace, service, name, err := splitSecretRef(ref)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	repoRoot, configPath, err := resolveRepoRoot(secretRmFlags.sopsConfig, cwd)
	if err != nil {
		return err
	}

	env, _, err := resolveEnvironment(configPath, secretRmFlags.env)
	if err != nil {
		return err
	}

	relPath := resolveSecretPath(secretRmFlags.secretsRoot, env, namespace, service, name)
	absPath := filepath.Join(repoRoot, filepath.FromSlash(relPath))

	if _, statErr := os.Stat(absPath); statErr != nil {
		if os.IsNotExist(statErr) {
			return fmt.Errorf("no secret at %s", relPath)
		}
		return statErr
	}
	if err := os.Remove(absPath); err != nil {
		return fmt.Errorf("remove %s: %w", relPath, err)
	}

	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "Removed %s\n", relPath)
	fmt.Fprintf(w, "\nNext: git add %s && git commit && git push\n", relPath)
	return nil
}
