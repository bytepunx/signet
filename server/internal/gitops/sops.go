package gitops

import (
	"errors"
	"fmt"

	"filippo.io/age"
	sops "github.com/getsops/sops/v3"
	sopsaes "github.com/getsops/sops/v3/aes"
	sopsage "github.com/getsops/sops/v3/age"
	sopsconfig "github.com/getsops/sops/v3/config"
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
	store := sopsyaml.NewStore(&sopsconfig.YAMLStoreConfig{})
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

	// Recover the SOPS data key by decrypting directly against each age
	// master key in the metadata.
	//
	// tree.Metadata.GetDataKey() is deliberately not used here: it routes
	// decryption through sops's local keyservice, which re-derives a fresh
	// age.MasterKey from just the wire-serialized recipient string (see
	// keyservice.Server.decryptWithAge) rather than using the *MasterKey
	// objects in tree.Metadata.KeyGroups. Any identities injected via
	// ParsedIdentities.ApplyToMasterKey are attached to those objects, so
	// GetDataKey() would silently ignore them and fall back to sops's normal
	// identity discovery (SOPS_AGE_KEY, ~/.ssh/id_rsa, etc.) — none of which
	// signet ever populates, since the private key must never touch disk or
	// the environment.
	dataKey, err := decryptDataKeyDirect(tree.Metadata, ids)
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

// decryptDataKeyDirect recovers the SOPS data key by calling Decrypt()
// directly on each age master key in md, injecting ids first. signet secret
// files are always encrypted to a flat group of age recipients (never
// PGP/KMS, never Shamir-sharded across multiple groups), so the first master
// key any provided identity can decrypt is authoritative.
func decryptDataKeyDirect(md sops.Metadata, ids sopsage.ParsedIdentities) ([]byte, error) {
	for _, group := range md.KeyGroups {
		for _, key := range group {
			mk, ok := key.(*sopsage.MasterKey)
			if !ok {
				continue
			}
			ids.ApplyToMasterKey(mk)
			if dataKey, err := mk.Decrypt(); err == nil {
				return dataKey, nil
			}
		}
	}
	return nil, errors.New("no provided age identity could decrypt the sops data key")
}
