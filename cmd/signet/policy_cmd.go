package main

import (
	"context"
	"fmt"
	"strings"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/spf13/cobra"
)

func init() {
	policyCmd.AddCommand(policyCreateCmd, policyListCmd, policyRemoveCmd)
	rootCmd.AddCommand(policyCmd)
}

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage cross-namespace/cross-service secret access policies",
	Long: "Manage access policies. Most workloads never need one — a workload whose\n" +
		"SPIFFE ID encodes ns/<namespace>/sa/<service> matching a secret's own\n" +
		"namespace/service is granted access automatically. Policies are only for\n" +
		"cross-namespace, cross-service, or wildcard grants. See docs/policies.md.",
}

var (
	policySpiffeID   string
	policyNamespace  string
	policyService    string
	policySecretName string
	policyPermission []string
)

var policyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Grant a SPIFFE ID (or glob) access to a namespace/service",
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := context.Background()
		conn, err := dialAdmin()
		if err != nil {
			return err
		}
		defer conn.Close()

		resp, err := adminClient(conn).CreatePolicy(ctx, &adminv1.CreatePolicyRequest{
			SpiffeId:    policySpiffeID,
			Namespace:   policyNamespace,
			Service:     policyService,
			SecretName:  policySecretName,
			Permissions: policyPermission,
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Policy created: %s\n", resp.GetId())
		return nil
	},
}

var policyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all access policies",
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := context.Background()
		conn, err := dialAdmin()
		if err != nil {
			return err
		}
		defer conn.Close()

		resp, err := adminClient(conn).ListPolicies(ctx, &adminv1.ListPoliciesRequest{})
		if err != nil {
			return err
		}
		w := cmd.OutOrStdout()
		if len(resp.GetPolicies()) == 0 {
			fmt.Fprintln(w, "No policies found. Most workloads don't need one — see docs/policies.md.")
			return nil
		}
		fmt.Fprintf(w, "%-12s %-45s %-12s %-25s %s\n", "ID", "SPIFFE ID", "NAMESPACE", "PATTERN", "PERMISSIONS")
		for _, p := range resp.GetPolicies() {
			fmt.Fprintf(w, "%-12s %-45s %-12s %-25s %s\n",
				p.GetId(), p.GetSpiffeId(), p.GetNamespace(), p.GetPattern(),
				strings.Join(p.GetPermissions(), ","))
		}
		return nil
	},
}

var policyRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Delete a policy by ID",
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := context.Background()
		conn, err := dialAdmin()
		if err != nil {
			return err
		}
		defer conn.Close()

		resp, err := adminClient(conn).DeletePolicy(ctx, &adminv1.DeletePolicyRequest{Id: policyID})
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), resp.GetMessage())
		return nil
	},
}

var policyID string

func init() {
	policyCreateCmd.Flags().StringVar(&policySpiffeID, "spiffe-id", "", "caller SPIFFE ID or glob, e.g. spiffe://td/ns/*/sa/echo (required)")
	policyCreateCmd.Flags().StringVar(&policyNamespace, "namespace", "", "target secret namespace, or \"*\" for any (required)")
	policyCreateCmd.Flags().StringVar(&policyService, "service", "", "target secret service (required)")
	policyCreateCmd.Flags().StringVar(&policySecretName, "secret-name", "", "secret name glob within namespace/service (default \"*\")")
	policyCreateCmd.Flags().StringArrayVar(&policyPermission, "permission", nil, "permission to grant, repeatable (default \"get\"); \"*\" grants all")
	_ = policyCreateCmd.MarkFlagRequired("spiffe-id")
	_ = policyCreateCmd.MarkFlagRequired("namespace")
	_ = policyCreateCmd.MarkFlagRequired("service")

	policyRemoveCmd.Flags().StringVar(&policyID, "id", "", "policy ID to remove (required)")
	_ = policyRemoveCmd.MarkFlagRequired("id")
}
