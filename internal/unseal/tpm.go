//go:build tpm

package unseal

import "fmt"

// UnsealWithTPM unseals the master key using the TPM 2.0 or vTPM device at
// devicePath (e.g. "/dev/tpmrm0"). The key must have been previously sealed to
// the TPM's current PCR state via the signet init process.
//
// If the TPM is unavailable or PCR measurements do not match, the server
// hard-fails rather than falling back to manual unsealing. Pass --tpm-fallback
// to permit manual unsealing when TPM fails.
func (m *Manager) UnsealWithTPM(devicePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state == StateUnsealed {
		return ErrAlreadyUnsealed
	}

	// TODO: implement TPM 2.0 unseal using github.com/google/go-tpm.
	// Steps:
	//   1. Open TPM device at devicePath
	//   2. Read the sealed key blob from well-known location
	//   3. Unseal against current PCR values
	//   4. Load the recovered key into m.store
	//   5. Transition to StateUnsealed
	return fmt.Errorf("TPM unseal: not yet implemented")
}
