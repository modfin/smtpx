package envelope

import (
	"bytes"
	"strings"
	"testing"
)

func TestDataStructure(t *testing.T) {
	t.Run("Prepend", func(t *testing.T) {
		d := &Data{}

		// Test prepending bytes
		n, err := d.Prepend([]byte("world"))
		if err != nil {
			t.Fatalf("Prepend failed: %v", err)
		}
		if n != 5 {
			t.Errorf("Expected to write 5 bytes, got %d", n)
		}

		// Test prepending string
		n, err = d.PrependString("hello ")
		if err != nil {
			t.Fatalf("PrependString failed: %v", err)
		}
		if n != 6 {
			t.Errorf("Expected to write 6 bytes, got %d", n)
		}

		// Verify correct order
		if d.String() != "hello world" {
			t.Errorf("Expected 'hello world', got '%s'", d.String())
		}
	})

	t.Run("AppendAndPrepend", func(t *testing.T) {
		d := &Data{}

		// Write to tail
		d.WriteString("middle")

		// Prepend to head
		d.PrependString("start ")

		// Append to tail
		d.WriteString(" end")

		if d.String() != "start middle end" {
			t.Errorf("Expected 'start middle end', got '%s'", d.String())
		}
	})

	t.Run("Len", func(t *testing.T) {
		d := &Data{}
		if d.Len() != 0 {
			t.Errorf("Expected length 0, got %d", d.Len())
		}

		d.WriteString("hello")
		if d.Len() != 5 {
			t.Errorf("Expected length 5, got %d", d.Len())
		}

		d.PrependString("12345")
		if d.Len() != 10 {
			t.Errorf("Expected length 10, got %d", d.Len())
		}
	})

	t.Run("Bytes", func(t *testing.T) {
		d := &Data{}
		d.WriteString("hello")
		d.PrependString("world")

		expected := []byte("worldhello")
		result := d.Bytes()

		if !bytes.Equal(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("ReadFrom", func(t *testing.T) {
		d := &Data{}
		reader := strings.NewReader("test data")

		n, err := d.ReadFrom(reader)
		if err != nil {
			t.Fatalf("ReadFrom failed: %v", err)
		}
		if n != 9 {
			t.Errorf("Expected to read 9 bytes, got %d", n)
		}

		if d.String() != "test data" {
			t.Errorf("Expected 'test data', got '%s'", d.String())
		}
	})
}
