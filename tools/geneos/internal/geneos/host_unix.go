//go:build !windows

package geneos

// SetWindowsReleaseEnv TODO: Document its behavior, inputs, and effects.
func (h *Host) SetWindowsReleaseEnv(osinfo map[string]string) (err error) {
	// This function is a no-op on non-Windows systems.
	// It is defined here to satisfy the interface but does not perform any actions.
	return
}
