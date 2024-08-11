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

func TestIsOutputPath(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		output string
		want   bool
	}{
		{
			name:   "matches same relative output",
			path:   "bundle.bundle",
			output: "bundle.bundle",
			want:   true,
		},
		{
			name:   "normalizes dot segments",
			path:   "./bundle.bundle",
			output: "bundle.bundle",
			want:   true,
		},
		{
			name:   "ignores surrounding whitespace",
			path:   " bundle.bundle ",
			output: "./bundle.bundle",
			want:   true,
		},
		{
			name:   "matches relative path against absolute output",
			path:   "bundle.bundle",
			output: filepath.Join(mustGetwd(t), "bundle.bundle"),
			want:   true,
		},
		{
			name:   "matches dot segments against absolute output",
			path:   "./tmp/../bundle.bundle",
			output: filepath.Join(mustGetwd(t), "bundle.bundle"),
			want:   true,
		},
		{
			name:   "does not match empty output",
			path:   "bundle.bundle",
			output: "",
			want:   false,
		},
		{
			name:   "does not match another file",
			path:   "input.txt",
			output: "bundle.bundle",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOutputPath(tt.path, tt.output); got != tt.want {
				t.Fatalf("isOutputPath(%q, %q) = %t, want %t", tt.path, tt.output, got, tt.want)
			}
		})
	}
}

func mustGetwd(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return wd
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
