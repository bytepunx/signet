package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	adminv1 "github.com/bytepunx/signet/gen/admin/v1"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var sopsKeyUpdateConfigFlags struct {
	file        string
	secretsPath string
	print       bool
}

func init() {
	sopsKeyUpdateConfigCmd.Flags().StringVar(&sopsKeyUpdateConfigFlags.file, "file", ".sops.yaml",
		"path to the .sops.yaml file to create or update")
	sopsKeyUpdateConfigCmd.Flags().StringVar(&sopsKeyUpdateConfigFlags.secretsPath, "secrets-path", "secrets/",
		"secrets directory prefix within the repository (e.g. secrets/)")
	sopsKeyUpdateConfigCmd.Flags().BoolVar(&sopsKeyUpdateConfigFlags.print, "print", false,
		"print the resulting config to stdout instead of writing to file")
	sopsKeyCmd.AddCommand(sopsKeyUpdateConfigCmd)
}

var sopsKeyUpdateConfigCmd = &cobra.Command{
	Use:   "update-config",
	Short: "Add or update this environment's age key in a .sops.yaml file",
	Long: `Queries the active age public key from signet and updates the .sops.yaml
file with:

  environments:
    <env>: <age-public-key>   # signet-specific; ignored by the sops CLI

  creation_rules:
    - signet_environment: <env>
      path_regex: ^secrets/<env>/
      age: <age-public-key>

Run this command once per environment against each environment's signet
instance to build a complete multi-environment .sops.yaml.  If the file
does not exist it is created; if a rule for this environment already
exists it is updated in place and all other content is preserved.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := context.Background()
		conn, err := dialAdmin(ctx)
		if err != nil {
			return err
		}
		defer conn.Close()

		resp, err := gitopsClient(conn).GetSOPSPublicKey(ctx, &adminv1.GetSOPSPublicKeyRequest{})
		if err != nil {
			return err
		}

		env := resp.GetEnvironment()
		pubKey := resp.GetPublicKey()

		secretsBase := strings.TrimRight(sopsKeyUpdateConfigFlags.secretsPath, "/")
		var pathRegex string
		if env != "" {
			pathRegex = fmt.Sprintf("^%s/%s/", secretsBase, env)
		} else {
			pathRegex = fmt.Sprintf("^%s/", secretsBase)
		}

		doc, err := loadSOPSDoc(sopsKeyUpdateConfigFlags.file)
		if err != nil {
			return fmt.Errorf("load %s: %w", sopsKeyUpdateConfigFlags.file, err)
		}

		root := docRoot(doc)

		if env != "" {
			setEnvKey(root, env, pubKey)
		}
		setCreationRule(root, env, pathRegex, pubKey)

		out, err := yaml.Marshal(doc)
		if err != nil {
			return fmt.Errorf("marshal yaml: %w", err)
		}

		if sopsKeyUpdateConfigFlags.print {
			_, err = cmd.OutOrStdout().Write(out)
			return err
		}

		if err := os.WriteFile(sopsKeyUpdateConfigFlags.file, out, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", sopsKeyUpdateConfigFlags.file, err)
		}

		w := cmd.OutOrStdout()
		fmt.Fprintf(w, "Updated %s\n", sopsKeyUpdateConfigFlags.file)
		if env != "" {
			fmt.Fprintf(w, "  environment: %s\n", env)
		} else {
			fmt.Fprintf(w, "  environment: (global)\n")
		}
		fmt.Fprintf(w, "  path_regex:  %s\n", pathRegex)
		fmt.Fprintf(w, "  age key:     %s\n", pubKey)
		return nil
	},
}

// loadSOPSDoc reads an existing .sops.yaml as a yaml.Document node, or
// returns a fresh empty document if the file does not exist.
func loadSOPSDoc(path string) (*yaml.Node, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		doc := &yaml.Node{
			Kind: yaml.DocumentNode,
			Content: []*yaml.Node{
				{Kind: yaml.MappingNode, Tag: "!!map"},
			},
		}
		return doc, nil
	}
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	if doc.Kind == 0 {
		doc = yaml.Node{
			Kind: yaml.DocumentNode,
			Content: []*yaml.Node{
				{Kind: yaml.MappingNode, Tag: "!!map"},
			},
		}
	}
	return &doc, nil
}

// docRoot returns the root mapping node of a DocumentNode.
func docRoot(doc *yaml.Node) *yaml.Node {
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return doc.Content[0]
	}
	return doc
}

// mappingGet finds a key in a YAML mapping node and returns its value node.
// Returns -1, nil if not found. Keys and values alternate in Content.
func mappingGet(m *yaml.Node, key string) (idx int, val *yaml.Node) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return i, m.Content[i+1]
		}
	}
	return -1, nil
}

// mappingSet sets key=val in a YAML mapping node, adding the pair if absent.
func mappingSet(m *yaml.Node, key string, val *yaml.Node) {
	idx, _ := mappingGet(m, key)
	if idx >= 0 {
		m.Content[idx+1] = val
		return
	}
	m.Content = append(m.Content,
		scalarNode(key),
		val,
	)
}

// setEnvKey sets environments.<env>=pubKey in the root mapping, creating
// the environments map if it does not exist.
func setEnvKey(root *yaml.Node, env, pubKey string) {
	_, envMap := mappingGet(root, "environments")
	if envMap == nil {
		envMap = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		mappingSet(root, "environments", envMap)
	}
	mappingSet(envMap, env, scalarNode(pubKey))
}

// setCreationRule finds the creation_rules sequence in root, then
// updates the entry for env or appends a new one.
func setCreationRule(root *yaml.Node, env, pathRegex, pubKey string) {
	_, seq := mappingGet(root, "creation_rules")
	if seq == nil {
		seq = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		mappingSet(root, "creation_rules", seq)
	}

	// Find an existing rule managed by signet for this environment.
	for _, ruleNode := range seq.Content {
		if ruleNode.Kind != yaml.MappingNode {
			continue
		}
		_, signetEnvNode := mappingGet(ruleNode, "signet_environment")
		existingEnv := ""
		if signetEnvNode != nil {
			existingEnv = signetEnvNode.Value
		}
		if existingEnv == env {
			mappingSet(ruleNode, "path_regex", scalarNode(pathRegex))
			mappingSet(ruleNode, "age", scalarNode(pubKey))
			return
		}
	}

	// No existing rule for this environment — build a new mapping node.
	// Put signet_environment first so it reads as a label.
	rule := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if env != "" {
		rule.Content = append(rule.Content,
			scalarNode("signet_environment"), scalarNode(env),
		)
	}
	rule.Content = append(rule.Content,
		scalarNode("path_regex"), scalarNode(pathRegex),
		scalarNode("age"), scalarNode(pubKey),
	)
	seq.Content = append(seq.Content, rule)
}

func scalarNode(val string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: val}
}
