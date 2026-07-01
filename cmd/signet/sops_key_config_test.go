package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// roundtrip marshals a yaml.Node to bytes, unmarshals into a map, and returns
// the map for easy assertion without duplicating yaml.Marshal boilerplate.
func roundtrip(t *testing.T, doc *yaml.Node) map[string]interface{} {
	t.Helper()
	out, err := yaml.Marshal(doc)
	require.NoError(t, err)
	var m map[string]interface{}
	require.NoError(t, yaml.Unmarshal(out, &m))
	return m
}

// TestUpdateConfig_FreshFile verifies that a non-existent file produces a
// well-formed .sops.yaml with an environments map and a creation_rules entry.
func TestUpdateConfig_FreshFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".sops.yaml")
	// File does not exist yet.

	doc, err := loadSOPSDoc(path)
	require.NoError(t, err)
	root := docRoot(doc)

	setEnvKey(root, "prod", "age1prod111")
	setCreationRule(root, "prod", "^secrets/prod/", "age1prod111")

	m := roundtrip(t, doc)

	envs, ok := m["environments"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "age1prod111", envs["prod"])

	rules, ok := m["creation_rules"].([]interface{})
	require.True(t, ok)
	require.Len(t, rules, 1)
	rule := rules[0].(map[string]interface{})
	assert.Equal(t, "prod", rule["signet_environment"])
	assert.Equal(t, "^secrets/prod/", rule["path_regex"])
	assert.Equal(t, "age1prod111", rule["age"])
}

// TestUpdateConfig_AddSecondEnvironment verifies that adding a second
// environment appends a new rule and leaves the first rule unchanged.
func TestUpdateConfig_AddSecondEnvironment(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".sops.yaml")
	initial := `environments:
    prod: age1prod111
creation_rules:
    - signet_environment: prod
      path_regex: ^secrets/prod/
      age: age1prod111
`
	require.NoError(t, os.WriteFile(path, []byte(initial), 0o644))

	doc, err := loadSOPSDoc(path)
	require.NoError(t, err)
	root := docRoot(doc)

	setEnvKey(root, "staging", "age1staging222")
	setCreationRule(root, "staging", "^secrets/staging/", "age1staging222")

	m := roundtrip(t, doc)

	envs := m["environments"].(map[string]interface{})
	assert.Equal(t, "age1prod111", envs["prod"])
	assert.Equal(t, "age1staging222", envs["staging"])

	rules := m["creation_rules"].([]interface{})
	require.Len(t, rules, 2)

	// Original rule must be intact.
	r0 := rules[0].(map[string]interface{})
	assert.Equal(t, "prod", r0["signet_environment"])
	assert.Equal(t, "age1prod111", r0["age"])

	r1 := rules[1].(map[string]interface{})
	assert.Equal(t, "staging", r1["signet_environment"])
	assert.Equal(t, "age1staging222", r1["age"])
}

// TestUpdateConfig_RotateKey verifies that updating an existing environment's
// rule replaces the age key without duplicating the rule.
func TestUpdateConfig_RotateKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".sops.yaml")
	initial := `environments:
    prod: age1prod111
creation_rules:
    - signet_environment: prod
      path_regex: ^secrets/prod/
      age: age1prod111
`
	require.NoError(t, os.WriteFile(path, []byte(initial), 0o644))

	doc, err := loadSOPSDoc(path)
	require.NoError(t, err)
	root := docRoot(doc)

	setEnvKey(root, "prod", "age1prod222new")
	setCreationRule(root, "prod", "^secrets/prod/", "age1prod222new")

	m := roundtrip(t, doc)

	envs := m["environments"].(map[string]interface{})
	assert.Equal(t, "age1prod222new", envs["prod"])

	rules := m["creation_rules"].([]interface{})
	require.Len(t, rules, 1, "rotation must not duplicate the rule")
	assert.Equal(t, "age1prod222new", rules[0].(map[string]interface{})["age"])
}

// TestUpdateConfig_PreservesNonSignetRules verifies that existing rules
// without a signet_environment annotation are preserved untouched.
func TestUpdateConfig_PreservesNonSignetRules(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".sops.yaml")
	initial := `creation_rules:
    - path_regex: ^terraform/
      age: age1terraform999
`
	require.NoError(t, os.WriteFile(path, []byte(initial), 0o644))

	doc, err := loadSOPSDoc(path)
	require.NoError(t, err)
	root := docRoot(doc)

	setEnvKey(root, "prod", "age1prod111")
	setCreationRule(root, "prod", "^secrets/prod/", "age1prod111")

	m := roundtrip(t, doc)

	rules := m["creation_rules"].([]interface{})
	require.Len(t, rules, 2)

	r0 := rules[0].(map[string]interface{})
	assert.Equal(t, "^terraform/", r0["path_regex"])
	assert.Equal(t, "age1terraform999", r0["age"])
	assert.Nil(t, r0["signet_environment"], "non-signet rule must not gain the annotation")

	r1 := rules[1].(map[string]interface{})
	assert.Equal(t, "prod", r1["signet_environment"])
}

// TestUpdateConfig_GlobalKey verifies that a global (no-environment) key
// generates a rule without the signet_environment annotation.
func TestUpdateConfig_GlobalKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".sops.yaml")

	doc, err := loadSOPSDoc(path)
	require.NoError(t, err)
	root := docRoot(doc)

	// Global rule: no environment set.
	setCreationRule(root, "", "^secrets/", "age1global000")

	m := roundtrip(t, doc)

	rules := m["creation_rules"].([]interface{})
	require.Len(t, rules, 1)
	rule := rules[0].(map[string]interface{})
	assert.Equal(t, "^secrets/", rule["path_regex"])
	assert.Equal(t, "age1global000", rule["age"])
	_, hasEnvKey := rule["signet_environment"]
	assert.False(t, hasEnvKey, "global rules must not have signet_environment")
}

// TestUpdateConfig_LoadSOPSDoc_MissingFile verifies that a missing file
// returns an empty document without error.
func TestUpdateConfig_LoadSOPSDoc_MissingFile(t *testing.T) {
	doc, err := loadSOPSDoc(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	require.NoError(t, err)
	require.NotNil(t, doc)
	assert.Equal(t, yaml.DocumentNode, doc.Kind)
}

// TestUpdateConfig_SignetEnvironmentFirst verifies that the signet_environment
// key appears as the first key in a newly created rule (readability).
func TestUpdateConfig_SignetEnvironmentFirst(t *testing.T) {
	doc, err := loadSOPSDoc(filepath.Join(t.TempDir(), ".sops.yaml"))
	require.NoError(t, err)
	root := docRoot(doc)

	setCreationRule(root, "dev", "^secrets/dev/", "age1dev333")

	out, err := yaml.Marshal(doc)
	require.NoError(t, err)
	yaml := string(out)

	idxEnv := strings.Index(yaml, "signet_environment")
	idxPath := strings.Index(yaml, "path_regex")
	assert.Greater(t, idxPath, idxEnv, "signet_environment must appear before path_regex")
}
