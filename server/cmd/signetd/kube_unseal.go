package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/bytepunx/signet/internal/api"
	"github.com/bytepunx/signet/internal/crypto"
	"github.com/bytepunx/signet/internal/store"
	"github.com/bytepunx/signet/internal/unseal"
)

// attemptKubeUnseal fetches the master key from a Kubernetes Secret and calls
// UnsealWithKey. It never fatals — any error is logged at WARN level and the
// server remains sealed so manual unseal is still possible. st and keyStore
// back the same key-check verification the manual/Shamir unseal RPCs use, so
// an auto-unseal with a stale or mismatched key is rejected and re-sealed
// rather than silently starting the server with the wrong master key.
func attemptKubeUnseal(ctx context.Context, mgr *unseal.Manager, st *store.Store, keyStore *crypto.KeyStore, secretName string) {
	if mgr.Status().State == unseal.StateUnsealed {
		slog.Info("kube-unseal: server already unsealed, skipping")
		return
	}

	namespace, err := podNamespace()
	if err != nil {
		slog.Warn("kube-unseal: cannot determine pod namespace", "err", err)
		return
	}

	client, err := inClusterClient()
	if err != nil {
		slog.Warn("kube-unseal: failed to build in-cluster client", "err", err)
		return
	}

	slog.Info("kube-unseal: fetching master key", "secret", fmt.Sprintf("%s/%s", namespace, secretName))

	sec, err := client.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		slog.Warn("kube-unseal: failed to fetch Secret", "secret", secretName, "namespace", namespace, "err", err)
		return
	}

	keyBytes, ok := sec.Data["master.key"]
	if !ok {
		slog.Warn("kube-unseal: Secret has no 'master.key' field", "secret", secretName, "namespace", namespace)
		return
	}

	// Copy into a fresh slice; UnsealWithKey (via KeyStore.Set) zeroes it.
	key := make([]byte, len(keyBytes))
	copy(key, keyBytes)

	if err = mgr.UnsealWithKey(key); err != nil {
		slog.Warn("kube-unseal: UnsealWithKey failed", "err", err)
		return
	}

	if err := api.VerifyOrInitKeyCheckValue(ctx, st, keyStore); err != nil {
		slog.Warn("kube-unseal: key check failed; the Secret's key does not match this deployment, re-sealing", "err", err)
		mgr.Seal()
		return
	}

	slog.Info("kube-unseal: server unsealed via Kubernetes Secret", "secret", fmt.Sprintf("%s/%s", namespace, secretName))
}

// podNamespace returns the namespace of the running pod by reading the
// Kubernetes-mounted service account namespace file.
func podNamespace() (string, error) {
	const nsFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	data, err := os.ReadFile(nsFile)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", nsFile, err)
	}
	ns := strings.TrimSpace(string(data))
	if ns == "" {
		return "", fmt.Errorf("%s is empty", nsFile)
	}
	return ns, nil
}

// inClusterClient builds a Kubernetes client using the in-cluster service
// account token and CA certificate.
func inClusterClient() (kubernetes.Interface, error) {
	rcfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("in-cluster config: %w", err)
	}
	client, err := kubernetes.NewForConfig(rcfg)
	if err != nil {
		return nil, fmt.Errorf("build client: %w", err)
	}
	return client, nil
}
