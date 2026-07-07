package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	flagInitNamespace   string
	flagInitKeySecret   string
	flagInitKubeContext string
	flagInitForce       bool
	flagInitDryRun      bool
	flagInitYes         bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap cluster for single-key operation and unseal signet",
	Long: `Prepare the cluster for single-key (non-Shamir) operation and unseal signet.

signet init:
  1. Checks whether signet is already unsealed (exits 0 if so).
  2. Looks for an existing master key in --key-secret / --namespace.
  3. Creates the Secret with a random 32-byte AES-256 key if it does not exist
     (or regenerates with --force).
  4. Calls the admin API to unseal signet with the key from the Secret.
  5. Verifies the server is now unsealed.

The Kubernetes Secret stores the raw 32-byte key in the 'master.key' field.
Access to this Secret grants access to the master key — restrict it with
Kubernetes RBAC and consider Sealed Secrets or External Secrets for
production deployments.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()

		conn, err := dialAdmin()
		if err != nil {
			return err
		}
		defer conn.Close()

		k8sClient, err := buildCLIK8sClient(flagInitKubeContext)
		if err != nil {
			return fmt.Errorf("kubernetes client: %w", err)
		}

		return runInitWithDeps(ctx, cmd.InOrStdin(), cmd.OutOrStdout(), adminClient(conn), k8sClient,
			flagInitNamespace, flagInitKeySecret, flagInitForce, flagInitDryRun, flagInitYes)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&flagInitNamespace, "namespace", "n", "signet",
		"Kubernetes namespace where signet is deployed")
	initCmd.Flags().StringVar(&flagInitKeySecret, "key-secret", "signet-master-key",
		"Name of the Kubernetes Secret to create or read")
	initCmd.Flags().StringVar(&flagInitKubeContext, "kube-context", "",
		"kubectl context to use (default: current context)")
	initCmd.Flags().BoolVar(&flagInitForce, "force", false,
		"Regenerate the master key and overwrite the existing Secret (DESTRUCTIVE: orphans every secret encrypted under the old key; requires confirmation unless --yes is set)")
	initCmd.Flags().BoolVar(&flagInitDryRun, "dry-run", false,
		"Print what would happen without making any changes")
	initCmd.Flags().BoolVar(&flagInitYes, "yes", false,
		"skip the interactive confirmation prompt required by --force")
}

// runInitWithDeps is the testable core of signet init. Real-dependency wiring
// happens in the cobra RunE above; tests inject fakes here directly.
func runInitWithDeps(
	ctx context.Context,
	in io.Reader,
	out io.Writer,
	admin adminv1.AdminServiceClient,
	k8s kubernetes.Interface,
	namespace, keySecret string,
	force, dryRun, skipConfirm bool,
) error {
	// 1. Check current seal state.
	stateResp, err := admin.Status(ctx, &adminv1.StatusRequest{})
	if err != nil {
		return fmt.Errorf("Status: %w", err)
	}
	if stateResp.State == adminv1.StatusResponse_STATE_UNSEALED {
		fmt.Fprintln(out, "signet is already unsealed.")
		return nil
	}
	fmt.Fprintf(out, "signet state: %s\n", stateName(stateResp.State))

	// 2. Resolve master key (read existing Secret or generate new one).
	key, action, err := manageKeySecret(ctx, in, out, k8s, namespace, keySecret, force, dryRun, skipConfirm)
	if err != nil {
		return err
	}
	if dryRun {
		fmt.Fprintf(out, "[dry-run] would unseal signet with key from Secret %s/%s.\n", namespace, keySecret)
		fmt.Fprintln(out, "[dry-run] no changes made.")
		return nil
	}

	// Zero key bytes on return regardless of outcome.
	defer func() {
		for i := range key {
			key[i] = 0
		}
	}()

	// 3. Unseal.
	if _, err = admin.UnsealKey(ctx, &adminv1.UnsealKeyRequest{Key: key}); err != nil {
		return fmt.Errorf("UnsealKey: %w", err)
	}

	// 4. Verify.
	stateResp, err = admin.Status(ctx, &adminv1.StatusRequest{})
	if err != nil {
		return fmt.Errorf("Status (verify): %w", err)
	}
	fmt.Fprintf(out, "signet unsealed using %s key from Secret %s/%s. State: %s\n",
		action, namespace, keySecret, stateName(stateResp.State))
	return nil
}

// manageKeySecret creates, reads, or overwrites the master key Secret and
// returns a fresh copy of the key bytes along with an action label ("created",
// "regenerated", or "existing"). The caller is responsible for zeroing the
// returned slice.
func manageKeySecret(
	ctx context.Context,
	in io.Reader,
	out io.Writer,
	k8s kubernetes.Interface,
	namespace, secretName string,
	force, dryRun, skipConfirm bool,
) (key []byte, action string, err error) {
	existing, getErr := k8s.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	notFound := k8serrors.IsNotFound(getErr)
	if getErr != nil && !notFound {
		return nil, "", fmt.Errorf("get Secret %s/%s: %w", namespace, secretName, getErr)
	}

	switch {
	case notFound:
		key, err = genKey()
		if err != nil {
			return nil, "", err
		}
		fmt.Fprintf(out, "Secret %q not found in namespace %q — generating new 32-byte master key.\n", secretName, namespace)
		if dryRun {
			fmt.Fprintf(out, "[dry-run] would create Secret %s/%s with new master key.\n", namespace, secretName)
			return key, "created", nil
		}
		sec := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: namespace},
			Type:       corev1.SecretTypeOpaque,
			Data:       map[string][]byte{"master.key": key},
		}
		if _, err = k8s.CoreV1().Secrets(namespace).Create(ctx, sec, metav1.CreateOptions{}); err != nil {
			return nil, "", fmt.Errorf("create Secret %s/%s: %w", namespace, secretName, err)
		}
		fmt.Fprintf(out, "Created Secret %s/%s.\n", namespace, secretName)
		printKeyWarning(out, namespace, secretName)
		return key, "created", nil

	case force:
		// Regenerating an EXISTING key orphans every secret currently
		// wrapped under the old one — this is destructive and, unlike the
		// notFound branch above, cannot be undone by re-running signet init.
		if !dryRun {
			warning := fmt.Sprintf(
				"WARNING: --force will overwrite Secret %s/%s with a new master key.\n"+
					"Every secret currently encrypted under the existing key will become\n"+
					"permanently undecryptable once the new key replaces it.", namespace, secretName)
			if err := confirmDestructive(in, out, warning, skipConfirm); err != nil {
				return nil, "", err
			}
		}
		key, err = genKey()
		if err != nil {
			return nil, "", err
		}
		fmt.Fprintf(out, "--force: regenerating master key for Secret %s/%s.\n", namespace, secretName)
		if dryRun {
			fmt.Fprintf(out, "[dry-run] would update Secret %s/%s with new master key.\n", namespace, secretName)
			return key, "regenerated", nil
		}
		existing.Data = map[string][]byte{"master.key": key}
		if _, err = k8s.CoreV1().Secrets(namespace).Update(ctx, existing, metav1.UpdateOptions{}); err != nil {
			return nil, "", fmt.Errorf("update Secret %s/%s: %w", namespace, secretName, err)
		}
		fmt.Fprintf(out, "Updated Secret %s/%s with new master key.\n", namespace, secretName)
		printKeyWarning(out, namespace, secretName)
		return key, "regenerated", nil

	default: // exists && !force
		keyBytes, ok := existing.Data["master.key"]
		if !ok {
			return nil, "", fmt.Errorf("Secret %s/%s has no 'master.key' field; use --force to regenerate", namespace, secretName)
		}
		if len(keyBytes) != 32 {
			return nil, "", fmt.Errorf("Secret %s/%s: 'master.key' is %d bytes, want 32; use --force to regenerate", namespace, secretName, len(keyBytes))
		}
		key = make([]byte, 32)
		copy(key, keyBytes)
		fmt.Fprintf(out, "Using existing master key from Secret %s/%s.\n", namespace, secretName)
		return key, "existing", nil
	}
}

func genKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generate master key: %w", err)
	}
	return key, nil
}

func printKeyWarning(out io.Writer, namespace, secretName string) {
	fmt.Fprintf(out, "WARNING: Secret %s/%s contains the plaintext master key.\n", namespace, secretName)
	fmt.Fprintln(out, "         Restrict access with Kubernetes RBAC. For production, consider")
	fmt.Fprintln(out, "         encrypting this Secret with Sealed Secrets or External Secrets.")
}

// buildCLIK8sClient builds a Kubernetes client from the default kubeconfig
// chain ($KUBECONFIG → ~/.kube/config), with an optional context override.
func buildCLIK8sClient(kubeContext string) (kubernetes.Interface, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}
	if kubeContext != "" {
		overrides.CurrentContext = kubeContext
	}
	restCfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules, overrides,
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}
	return kubernetes.NewForConfig(restCfg)
}
