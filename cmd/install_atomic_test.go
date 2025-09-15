package cmd

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestInstallBinaryAtomic(t *testing.T) {
	// Create temp directories
	srcDir := t.TempDir()
	destDir := t.TempDir()

	// Create source binary
	srcPath := filepath.Join(srcDir, "binary")
	srcContent := []byte("test binary content")
	if err := os.WriteFile(srcPath, srcContent, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	destPath := filepath.Join(destDir, "installed-binary")

	t.Run("Atomic installation", func(t *testing.T) {
		// First installation
		err := installBinary(srcPath, destPath)
		if err != nil {
			t.Errorf("installBinary() error = %v", err)
		}

		// Verify file exists and is executable
		info, err := os.Stat(destPath)
		if err != nil {
			t.Fatalf("Failed to stat destination file: %v", err)
		}
		if info.Mode()&0755 != 0755 {
			t.Errorf("File is not executable: %v", info.Mode())
		}

		// Verify temp file is cleaned up
		entries, err := os.ReadDir(destDir)
		if err != nil {
			t.Fatalf("Failed to read directory: %v", err)
		}
		for _, entry := range entries {
			if entry.Name() != "installed-binary" && entry.Name()[:11] == ".binst-tmp-" {
				t.Errorf("Temporary file not cleaned up: %s", entry.Name())
			}
		}
	})

	t.Run("Overwrite existing binary atomically", func(t *testing.T) {
		// Create existing binary with different content
		existingContent := []byte("existing binary")
		if err := os.WriteFile(destPath, existingContent, 0755); err != nil {
			t.Fatalf("Failed to create existing file: %v", err)
		}

		// Install over existing file
		err := installBinary(srcPath, destPath)
		if err != nil {
			t.Errorf("installBinary() error = %v", err)
		}

		// Verify new content
		content, err := os.ReadFile(destPath)
		if err != nil {
			t.Fatalf("Failed to read destination file: %v", err)
		}
		if string(content) != string(srcContent) {
			t.Errorf("File content mismatch after overwrite")
		}
	})

	t.Run("Concurrent installations", func(t *testing.T) {
		// Test that concurrent installations don't interfere
		var wg sync.WaitGroup
		errors := make([]error, 5)

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				binPath := filepath.Join(destDir, "concurrent-binary")
				errors[idx] = installBinary(srcPath, binPath)
			}(i)
		}

		wg.Wait()

		// At least one should succeed
		successCount := 0
		for _, err := range errors {
			if err == nil {
				successCount++
			}
		}
		if successCount == 0 {
			t.Error("All concurrent installations failed")
		}

		// Verify final file is valid
		binPath := filepath.Join(destDir, "concurrent-binary")
		info, err := os.Stat(binPath)
		if err != nil {
			t.Fatalf("Failed to stat concurrent binary: %v", err)
		}
		if info.Mode()&0755 != 0755 {
			t.Errorf("Concurrent binary is not executable: %v", info.Mode())
		}
	})
}
