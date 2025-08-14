package process

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestRemoveArgs(t *testing.T) {
	// Test removing single argument
	args := []string{"arg1", "arg2", "arg3", "arg4"}
	remove := []string{"arg2"}
	result := RemoveArgs(args, remove...)
	expected := []string{"arg1", "arg3", "arg4"}

	if len(result) != len(expected) {
		t.Errorf("Expected %d args, got %d", len(expected), len(result))
	}

	for i, arg := range expected {
		if result[i] != arg {
			t.Errorf("Expected '%s' at index %d, got '%s'", arg, i, result[i])
		}
	}

	// Test removing multiple arguments
	args = []string{"arg1", "arg2", "arg3", "arg4", "arg5"}
	remove = []string{"arg2", "arg4"}
	result = RemoveArgs(args, remove...)
	expected = []string{"arg1", "arg3", "arg5"}

	if len(result) != len(expected) {
		t.Errorf("Expected %d args, got %d", len(expected), len(result))
	}

	for i, arg := range expected {
		if result[i] != arg {
			t.Errorf("Expected '%s' at index %d, got '%s'", arg, i, result[i])
		}
	}

	// Test removing non-existent argument
	args = []string{"arg1", "arg2", "arg3"}
	remove = []string{"nonexistent"}
	result = RemoveArgs(args, remove...)
	expected = []string{"arg1", "arg2", "arg3"}

	if len(result) != len(expected) {
		t.Errorf("Expected %d args, got %d", len(expected), len(result))
	}

	// Test removing from empty slice
	args = []string{}
	remove = []string{"arg1"}
	result = RemoveArgs(args, remove...)
	if len(result) != 0 {
		t.Errorf("Expected 0 args, got %d", len(result))
	}

	// Test removing all arguments
	args = []string{"arg1", "arg2", "arg3"}
	remove = []string{"arg1", "arg2", "arg3"}
	result = RemoveArgs(args, remove...)
	if len(result) != 0 {
		t.Errorf("Expected 0 args, got %d", len(result))
	}
}

func TestRetErrIfFalse(t *testing.T) {
	// Test with true condition
	err := retErrIfFalse(true, nil)
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}

	// Test with false condition and nil error
	err = retErrIfFalse(false, nil)
	if err == nil {
		t.Error("Expected error when condition is false")
	}

	// Test with false condition and existing error
	expectedErr := fmt.Errorf("test error")
	err = retErrIfFalse(false, expectedErr)
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestProgramValidation(t *testing.T) {
	program := Program{}

	// Test with empty executable
	_, err := Start(nil, program)
	if err == nil {
		t.Error("Expected error with empty executable")
	}

	// Test with valid executable but invalid working directory
	program.Executable = "echo"
	program.WorkingDir = "/nonexistent/directory"
	_, err = Start(nil, program)
	if err == nil {
		t.Error("Expected error with invalid working directory")
	}
}

func TestSliceFromAny(t *testing.T) {
	// Test with string slice
	input := []string{"arg1", "arg2", "arg3"}
	result, err := sliceFromAny(input)
	if err != nil {
		t.Errorf("sliceFromAny failed: %v", err)
	}

	if len(result) != len(input) {
		t.Errorf("Expected %d items, got %d", len(input), len(result))
	}

	for i, item := range input {
		if result[i] != item {
			t.Errorf("Expected '%s' at index %d, got '%s'", item, i, result[i])
		}
	}

	// Test with interface{} slice
	inputInterface := []interface{}{"arg1", "arg2", "arg3"}
	result, err = sliceFromAny(inputInterface)
	if err != nil {
		t.Errorf("sliceFromAny failed: %v", err)
	}

	if len(result) != len(inputInterface) {
		t.Errorf("Expected %d items, got %d", len(inputInterface), len(result))
	}

	// Test with unsupported type
	_, err = sliceFromAny(42)
	if err == nil {
		t.Error("Expected error with unsupported type")
	}

	// Test with nil
	_, err = sliceFromAny(nil)
	if err == nil {
		t.Error("Expected error with nil input")
	}
}

func TestWriterFromAny(t *testing.T) {
	// Test with string path
	dest := "/tmp/test_file"
	writer, err := writerFromAny(dest, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Errorf("writerFromAny failed: %v", err)
	}
	if writer == nil {
		t.Error("writerFromAny should not return nil writer")
	}

	// Test with invalid path
	dest = "/nonexistent/directory/test_file"
	_, err = writerFromAny(dest, os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		t.Error("Expected error with invalid path")
	}

	// Test with unsupported type
	_, err = writerFromAny(42, os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		t.Error("Expected error with unsupported type")
	}

	// Test with nil
	_, err = writerFromAny(nil, os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		t.Error("Expected error with nil input")
	}
}

func TestBatch(t *testing.T) {
	// Test with empty batch
	programs := []Program{}
	done, err := Batch(nil, programs)
	if err != nil {
		t.Errorf("Batch failed with empty programs: %v", err)
	}
	if done == nil {
		t.Error("Batch should return non-nil done channel")
	}

	// Test with single program
	programs = []Program{
		{
			Executable: "echo",
			Args:       []string{"hello"},
		},
	}
	done, err = Batch(nil, programs)
	if err != nil {
		t.Errorf("Batch failed with single program: %v", err)
	}
	if done == nil {
		t.Error("Batch should return non-nil done channel")
	}

	// Test with multiple programs
	programs = []Program{
		{
			Executable: "echo",
			Args:       []string{"hello"},
		},
		{
			Executable: "echo",
			Args:       []string{"world"},
		},
	}
	done, err = Batch(nil, programs)
	if err != nil {
		t.Errorf("Batch failed with multiple programs: %v", err)
	}
	if done == nil {
		t.Error("Batch should return non-nil done channel")
	}
}

func TestGetPID(t *testing.T) {
	// Test with empty binary prefix
	_, err := GetPID(nil, "", false, nil, nil)
	if err == nil {
		t.Error("Expected error with empty binary prefix")
	}

	// Test with custom check function
	customCheck := func(arg any, cmdline []string) bool {
		return true
	}
	_, err = GetPID(nil, "test", false, customCheck, nil)
	if err == nil {
		t.Error("Expected error with nil host")
	}
}

func TestProgramFields(t *testing.T) {
	program := Program{
		Executable: "test",
		Username:   "testuser",
		WorkingDir: "/tmp",
		ErrLog:     "test.log",
		Args:       []string{"arg1", "arg2"},
		Env:        []string{"KEY=value"},
		Foreground: false,
		Restart:    true,
		IgnoreErr:  false,
		Wait:       5 * time.Second,
	}

	// Test field access
	if program.Executable != "test" {
		t.Errorf("Expected executable 'test', got '%s'", program.Executable)
	}

	if program.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", program.Username)
	}

	if program.WorkingDir != "/tmp" {
		t.Errorf("Expected working dir '/tmp', got '%s'", program.WorkingDir)
	}

	if program.ErrLog != "test.log" {
		t.Errorf("Expected err log 'test.log', got '%s'", program.ErrLog)
	}

	if len(program.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(program.Args))
	}

	if len(program.Env) != 1 {
		t.Errorf("Expected 1 env var, got %d", len(program.Env))
	}

	if program.Foreground {
		t.Error("Expected foreground to be false")
	}

	if !program.Restart {
		t.Error("Expected restart to be true")
	}

	if program.IgnoreErr {
		t.Error("Expected ignore err to be false")
	}

	if program.Wait != 5*time.Second {
		t.Errorf("Expected wait 5s, got %v", program.Wait)
	}
}

func TestOptions(t *testing.T) {
	// Test default options
	options := []Options{}
	program := Program{
		Executable: "echo",
		Args:       []string{"hello"},
	}

	// This should not panic
	_, err := Start(nil, program, options...)
	if err == nil {
		t.Error("Expected error with nil host")
	}
}
