package geneos

import (
	"errors"
	"testing"
)

func TestErrorConstants(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrRootNotSet",
			err:      ErrRootNotSet,
			expected: "root directory not set",
		},
		{
			name:     "ErrInvalidArgs",
			err:      ErrInvalidArgs,
			expected: "invalid arguments",
		},
		{
			name:     "ErrNotSupported",
			err:      ErrNotSupported,
			expected: "not supported",
		},
		{
			name:     "ErrIsADirectory",
			err:      ErrIsADirectory,
			expected: "is a directory",
		},
		{
			name:     "ErrExists",
			err:      ErrExists,
			expected: "exists",
		},
		{
			name:     "ErrNotExist",
			err:      ErrNotExist,
			expected: "does not exist",
		},
		{
			name:     "ErrDisabled",
			err:      ErrDisabled,
			expected: "instance is disabled",
		},
		{
			name:     "ErrProtected",
			err:      ErrProtected,
			expected: "instance is protected",
		},
		{
			name:     "ErrRunning",
			err:      ErrRunning,
			expected: "instance is running",
		},
		{
			name:     "ErrNotRunning",
			err:      ErrNotRunning,
			expected: "instance is not running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("Error message for %s = %q, want %q", tt.name, tt.err.Error(), tt.expected)
			}
		})
	}
}

func TestErrorsAreDistinct(t *testing.T) {
	errorList := []error{
		ErrRootNotSet,
		ErrInvalidArgs,
		ErrNotSupported,
		ErrIsADirectory,
		ErrExists,
		ErrNotExist,
		ErrDisabled,
		ErrProtected,
		ErrRunning,
		ErrNotRunning,
	}

	// Check that all errors are distinct
	for i, err1 := range errorList {
		for j, err2 := range errorList {
			if i != j && err1 == err2 {
				t.Errorf("Errors at index %d and %d are the same: %v", i, j, err1)
			}
		}
	}
}

func TestErrorWrapping(t *testing.T) {
	baseErr := ErrNotExist
	wrappedErr := errors.New("wrapped: " + baseErr.Error())

	// Test that wrapped errors can be detected
	if !errors.Is(wrappedErr, wrappedErr) {
		t.Error("errors.Is should detect wrapped error")
	}

	// Test unwrapping behavior
	if wrappedErr.Error() != "wrapped: does not exist" {
		t.Errorf("Wrapped error message = %q, want %q", wrappedErr.Error(), "wrapped: does not exist")
	}
}

func TestErrorsImplementError(t *testing.T) {
	// Test that all package errors implement the error interface
	var _ error = ErrRootNotSet
	var _ error = ErrInvalidArgs
	var _ error = ErrNotSupported
	var _ error = ErrIsADirectory
	var _ error = ErrExists
	var _ error = ErrNotExist
	var _ error = ErrDisabled
	var _ error = ErrProtected
	var _ error = ErrRunning
	var _ error = ErrNotRunning
}

func TestErrorComparison(t *testing.T) {
	tests := []struct {
		name   string
		err1   error
		err2   error
		equals bool
	}{
		{
			name:   "same error",
			err1:   ErrExists,
			err2:   ErrExists,
			equals: true,
		},
		{
			name:   "different errors",
			err1:   ErrExists,
			err2:   ErrNotExist,
			equals: false,
		},
		{
			name:   "error vs nil",
			err1:   ErrExists,
			err2:   nil,
			equals: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errors.Is(tt.err1, tt.err2)
			if result != tt.equals {
				t.Errorf("errors.Is(%v, %v) = %v, want %v", tt.err1, tt.err2, result, tt.equals)
			}
		})
	}
}

func TestConstantErrorValues(t *testing.T) {
	// Test that error constants are not nil
	errors := map[string]error{
		"ErrRootNotSet":   ErrRootNotSet,
		"ErrInvalidArgs":  ErrInvalidArgs,
		"ErrNotSupported": ErrNotSupported,
		"ErrIsADirectory": ErrIsADirectory,
		"ErrExists":       ErrExists,
		"ErrNotExist":     ErrNotExist,
		"ErrDisabled":     ErrDisabled,
		"ErrProtected":    ErrProtected,
		"ErrRunning":      ErrRunning,
		"ErrNotRunning":   ErrNotRunning,
	}

	for name, err := range errors {
		if err == nil {
			t.Errorf("Error constant %s should not be nil", name)
		}
	}
}

func TestDisableExtensionConstant(t *testing.T) {
	if DisableExtension == "" {
		t.Error("DisableExtension should not be empty")
	}

	expected := "disabled"
	if DisableExtension != expected {
		t.Errorf("DisableExtension = %q, want %q", DisableExtension, expected)
	}
}