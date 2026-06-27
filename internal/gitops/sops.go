package gitops

import (
	"errors"
	"fmt"

	"filippo.io/age"
	sopsaes "github.com/getsops/sops/v3/aes"
	sopsage "github.com/getsops/sops/v3/age"
	sopsyaml "github.com/getsops/sops/v3/stores/yaml"
	"gopkg.in/yaml.v3"
)

// DecryptFile decrypts a SOPS-encrypted YAML file entirely in memory.
// It tries each provided identity against the age recipients embedded in the
// SOPS metadata until one succeeds.
//
// The file must contain a top-level "value" key; its string value is returned
// as the plaintext secret bytes.
func DecryptFile(data []byte, identities []age.Identity) ([]byte, error) {
	if len(identities) == 0 {
		return nil, errors.New("no age identities provided")
	}

	// Parse SOPS-encrypted YAML into a tree.
	store := sopsyaml.NewStore(nil)
	tree, err := store.LoadEncryptedFile(data)
	if err != nil {
		return nil, fmt.Errorf("parse sops file: %w", err)
	}

	// Convert age.Identity values to armored strings and load them into a
	// ParsedIdentities that we can inject into each MasterKey.
	var ids sopsage.ParsedIdentities
	for _, id := range identities {
		xi, ok := id.(*age.X25519Identity)
		if !ok {
			continue
		}
		if err := ids.Import(xi.String()); err != nil {
			return nil, fmt.Errorf("import age identity: %w", err)
		}
	}
	if len(ids) == 0 {
		return nil, errors.New("no X25519 age identities found in provided set")
	}

	// Inject identities into every age master key in the SOPS metadata.
	// GetDataKey() will try each one during decryption.
	for _, group := range tree.Metadata.KeyGroups {
		for _, key := range group {
			if mk, ok := key.(*sopsage.MasterKey); ok {
				ids.ApplyToMasterKey(mk)
			}
		}
	}

	// Recover the SOPS data key using the injected age identities.
	dataKey, err := tree.Metadata.GetDataKey()
	if err != nil {
		return nil, fmt.Errorf("decrypt sops data key: %w", err)
	}

	// Decrypt all encrypted leaf values in the tree.
	if _, err := tree.Decrypt(dataKey, sopsaes.NewCipher()); err != nil {
		return nil, fmt.Errorf("decrypt sops tree: %w", err)
	}

	// Emit the decrypted YAML bytes.
	plain, err := store.EmitPlainFile(tree.Branches)
	if err != nil {
		return nil, fmt.Errorf("emit plain sops file: %w", err)
	}

	// Extract the top-level "value" key.
	var doc struct {
		Value string `yaml:"value"`
	}
	if err := yaml.Unmarshal(plain, &doc); err != nil {
		return nil, fmt.Errorf("parse decrypted yaml: %w", err)
	}
	if doc.Value == "" {
		return nil, errors.New("decrypted yaml has no top-level \"value\" field")
	}

	return []byte(doc.Value), nil
}
