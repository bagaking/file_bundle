package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestProcessFileWritesBundleSectionAndUpdatesCounters(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	inputPath := "input.txt"
	if err := os.WriteFile(inputPath, []byte("  alpha  \n\n   \n beta\t\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	outFile, err := os.Create("bundle.bundle")
	if err != nil {
		t.Fatal(err)
	}

	originalShrink := shrink
	originalVerbose := verbose
	originalLineCount := lineCount
	originalCharCount := charCount
	originalFileCount := fileCount
	t.Cleanup(func() {
		shrink = originalShrink
		verbose = originalVerbose
		lineCount = originalLineCount
		charCount = originalCharCount
		fileCount = originalFileCount
	})

	shrink = true
	verbose = false
	lineCount = 0
	charCount = 0
	fileCount = 0

	processFile(inputPath, outFile, Config{Description: "release snapshot"})

	if err := outFile.Close(); err != nil {
		t.Fatal(err)
	}

	gotBytes, err := os.ReadFile("bundle.bundle")
	if err != nil {
		t.Fatal(err)
	}
	got := string(gotBytes)

	lines := strings.Split(got, "\n")
	if len(lines) != 10 {
		t.Fatalf("bundle output line count = %d, want 10:\n%s", len(lines), got)
	}
	wantLines := []string{
		"==========",
		"!! release snapshot",
		"File: input.txt",
	}
	for i, want := range wantLines {
		if lines[i] != want {
			t.Fatalf("bundle output line %d = %q, want %q:\n%s", i, lines[i], want, got)
		}
	}
	if !strings.HasPrefix(lines[3], "Time: ") {
		t.Fatalf("bundle time line = %q, want Time prefix:\n%s", lines[3], got)
	}
	if _, err := time.Parse("2006-01-02 15:04:05", strings.TrimPrefix(lines[3], "Time: ")); err != nil {
		t.Fatalf("bundle time line has invalid timestamp %q: %v", lines[3], err)
	}
	wantTail := []string{"==========", "alpha", "", "beta", "", ""}
	for i, want := range wantTail {
		lineIndex := i + 4
		if lines[lineIndex] != want {
			t.Fatalf("bundle output line %d = %q, want %q:\n%s", lineIndex, lines[lineIndex], want, got)
		}
	}
	if fileCount != 1 {
		t.Fatalf("fileCount = %d, want 1", fileCount)
	}
	if lineCount != 4 {
		t.Fatalf("lineCount = %d, want 4", lineCount)
	}
	if charCount != len("alpha\n\nbeta\n") {
		t.Fatalf("charCount = %d, want %d", charCount, len("alpha\n\nbeta\n"))
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
