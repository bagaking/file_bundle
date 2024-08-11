package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShrinkContent(t *testing.T) {
	tests := []struct {
		name       string
		shrinkMode bool
		input      string
		want       string
	}{
		{
			name:       "preserves content when disabled",
			shrinkMode: false,
			input:      "  alpha  \n\n\n beta\t\n",
			want:       "  alpha  \n\n\n beta\t\n",
		},
		{
			name:       "trims lines and collapses repeated empty lines",
			shrinkMode: true,
			input:      "  alpha  \n\n   \n beta\t\n",
			want:       "alpha\n\nbeta\n",
		},
		{
			name:       "keeps a single empty line between blocks",
			shrinkMode: true,
			input:      "first\n\n\nsecond",
			want:       "first\n\nsecond",
		},
	}

	originalShrink := shrink
	t.Cleanup(func() {
		shrink = originalShrink
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shrink = tt.shrinkMode

			got := shrinkContent([]byte(tt.input))
			if got != tt.want {
				t.Errorf("shrinkContent(%q) with shrink=%t = %q, want %q", tt.input, tt.shrinkMode, got, tt.want)
			}
		})
	}
}

func TestSeekConfFileName(t *testing.T) {
	t.Run("finds config in current directory", func(t *testing.T) {
		dir := t.TempDir()
		chdir(t, dir)

		if err := os.WriteFile("example.file_bundle_rc", []byte("entry = []\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir("nested", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join("nested", "ignored.file_bundle_rc"), []byte("entry = []\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		got, err := seekConfFileName()
		if err != nil {
			t.Fatalf("seekConfFileName() returned error: %v", err)
		}
		if got != "example.file_bundle_rc" {
			t.Fatalf("seekConfFileName() = %q, want %q", got, "example.file_bundle_rc")
		}
	})

	t.Run("returns not found when current directory has no config", func(t *testing.T) {
		dir := t.TempDir()
		chdir(t, dir)

		if _, err := seekConfFileName(); err == nil {
			t.Fatal("seekConfFileName() returned nil error, want not found error")
		}
	})
}

func chdir(t *testing.T, dir string) {
	t.Helper()

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatal(err)
		}
	})
}
