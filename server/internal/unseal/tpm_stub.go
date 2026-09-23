//go:build !tpm

package unseal

// UnsealWithTPM is not available in this build. Recompile with -tags tpm to
// enable TPM/vTPM auto-unsealing.
func (m *Manager) UnsealWithTPM(_ string) error {
	return ErrTPMNotSupported
}
