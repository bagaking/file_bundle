package main

import (
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
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

func TestIsWithinWorkingDirectory(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "project")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	chdir(t, dir)

	if err := os.WriteFile("input.txt", []byte("inside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outsidePath := filepath.Join(parent, "secret.txt")
	if err := os.WriteFile(outsidePath, []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "accepts normal inside path",
			path: "input.txt",
			want: true,
		},
		{
			name: "rejects empty path",
			path: "",
			want: false,
		},
		{
			name: "rejects parent path",
			path: "../secret.txt",
			want: false,
		},
		{
			name: "rejects outside absolute path",
			path: outsidePath,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isWithinWorkingDirectory(tt.path); got != tt.want {
				t.Fatalf("isWithinWorkingDirectory(%q) = %t, want %t", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsWithinWorkingDirectoryFailsClosedWhenWorkingDirectoryIsUnavailable(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "deleted")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	oldDir := mustGetwd(t)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatal(err)
		}
	})

	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	if got := isWithinWorkingDirectory("input.txt"); got {
		t.Fatal("isWithinWorkingDirectory() = true with unavailable cwd, want false")
	}
}

func TestIsOutputWithinWorkingDirectory(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "project")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	chdir(t, dir)

	if err := os.WriteFile("existing.bundle", []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir("out", 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "accepts new file in working directory",
			path: "bundle.bundle",
			want: true,
		},
		{
			name: "accepts existing file in working directory",
			path: "existing.bundle",
			want: true,
		},
		{
			name: "accepts new file in existing child directory",
			path: filepath.Join("out", "bundle.bundle"),
			want: true,
		},
		{
			name: "rejects empty path",
			path: "",
			want: false,
		},
		{
			name: "rejects parent directory output",
			path: "../bundle.bundle",
			want: false,
		},
		{
			name: "rejects output in missing child directory",
			path: filepath.Join("missing", "bundle.bundle"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOutputWithinWorkingDirectory(tt.path); got != tt.want {
				t.Fatalf("isOutputWithinWorkingDirectory(%q) = %t, want %t", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsOutputWithinWorkingDirectoryRejectsExternalSymlink(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "project")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	chdir(t, dir)

	externalPath := filepath.Join(parent, "external.bundle")
	if err := os.WriteFile(externalPath, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(externalPath, "bundle-link.bundle"); err != nil {
		t.Skipf("os.Symlink(%q, %q) error = %v; skipping symlink output boundary smoke", externalPath, "bundle-link.bundle", err)
	}

	if got := isOutputWithinWorkingDirectory("bundle-link.bundle"); got {
		t.Fatal("isOutputWithinWorkingDirectory() = true for external symlink output, want false")
	}
}

func TestValidateOutputWithinWorkingDirectoryExplainsMissingParent(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	output := filepath.Join("missing", "bundle.bundle")
	err := validateOutputWithinWorkingDirectory(output)
	if err == nil {
		t.Fatalf("validateOutputWithinWorkingDirectory(%q) error = nil, want error", output)
	}
	if !strings.Contains(err.Error(), "parent directory") {
		t.Fatalf("validateOutputWithinWorkingDirectory(%q) error = %q, want parent directory context", output, err)
	}
	if strings.Contains(err.Error(), "outside the working directory") {
		t.Fatalf("validateOutputWithinWorkingDirectory(%q) error = %q, want missing-parent message", output, err)
	}
}

func TestCLIRejectsOutputOutsideWorkingDirectory(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "project")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	chdir(t, dir)

	if err := os.WriteFile("input.txt", []byte("inside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	configPath := "paths.file_bundle_rc"
	writeConfig(t, configPath, Config{
		Entry:  []string{"input.txt"},
		Output: "../bundle.bundle",
	})

	status := runMainExpectingExit(t, "-i", configPath)
	if status != 1 {
		t.Fatalf("file_bundle -i paths.file_bundle_rc exit status = %d, want 1", status)
	}
	if _, err := os.Stat(filepath.Join(parent, "bundle.bundle")); !os.IsNotExist(err) {
		t.Fatalf("parent bundle stat error = %v, want not exist", err)
	}
}

func TestCLIRejectsEntryOutsideWorkingDirectory(t *testing.T) {
	tests := []struct {
		name         string
		entry        []string
		wantIncluded string
		wantSkipped  string
	}{
		{
			name:         "skips parent directory match",
			entry:        []string{"input.txt", "../secret.txt"},
			wantIncluded: "File: input.txt",
			wantSkipped:  "outside",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent := t.TempDir()
			dir := filepath.Join(parent, "project")
			if err := os.Mkdir(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			chdir(t, dir)

			if err := os.WriteFile(filepath.Join(parent, "secret.txt"), []byte("outside\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile("input.txt", []byte("inside\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			configPath := "paths.file_bundle_rc"
			config := Config{
				Entry:  tt.entry,
				Output: "bundle.bundle",
			}
			writeConfig(t, configPath, config)

			runMainForTest(t, "-i", configPath)

			gotBytes, err := os.ReadFile("bundle.bundle")
			if err != nil {
				t.Fatalf("os.ReadFile(%q) error = %v, want nil", "bundle.bundle", err)
			}
			got := string(gotBytes)
			if !strings.Contains(got, tt.wantIncluded) {
				t.Errorf("file_bundle -i paths.file_bundle_rc bundle contains %q = false, want true:\n%s", tt.wantIncluded, got)
			}
			if strings.Contains(got, tt.wantSkipped) {
				t.Errorf("file_bundle -i paths.file_bundle_rc bundle contains %q = true, want false:\n%s", tt.wantSkipped, got)
			}
			if fileCount != 1 {
				t.Errorf("fileCount after file_bundle -i paths.file_bundle_rc = %d, want 1", fileCount)
			}
		})
	}
}

func TestCLIRejectsSymlinkOutsideWorkingDirectory(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "project")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	chdir(t, dir)

	externalPath := filepath.Join(parent, "external.txt")
	if err := os.WriteFile(externalPath, []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("input.txt", []byte("inside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(externalPath, "external-link.txt"); err != nil {
		t.Skipf("os.Symlink(%q, %q) error = %v; skipping symlink boundary smoke", externalPath, "external-link.txt", err)
	}

	configPath := "links.file_bundle_rc"
	writeConfig(t, configPath, Config{
		Entry:  []string{"*"},
		Output: "bundle.bundle",
	})

	runMainForTest(t, "-i", configPath)

	gotBytes, err := os.ReadFile("bundle.bundle")
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil", "bundle.bundle", err)
	}
	got := string(gotBytes)
	if !strings.Contains(got, "File: input.txt") {
		t.Errorf("file_bundle -i links.file_bundle_rc bundle contains %q = false, want true:\n%s", "File: input.txt", got)
	}
	if strings.Contains(got, "outside") {
		t.Errorf("file_bundle -i links.file_bundle_rc bundle contains external symlink content = true, want false:\n%s", got)
	}
	if strings.Contains(got, "File: external-link.txt") {
		t.Errorf("file_bundle -i links.file_bundle_rc bundle contains symlink path = true, want false:\n%s", got)
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

func TestCLIExcludesOutputCreatedBeforeGlobExpansion(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	if err := os.WriteFile("input.txt", []byte("alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	configPath := "all.file_bundle_rc"
	config := Config{
		Entry:  []string{"*"},
		Output: "bundle.bundle",
	}
	writeConfig(t, configPath, config)

	runMainForTest(t, "-i", configPath)

	gotBytes, err := os.ReadFile("bundle.bundle")
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil", "bundle.bundle", err)
	}
	got := string(gotBytes)
	if !strings.Contains(got, "File: input.txt") {
		t.Errorf("file_bundle -i all.file_bundle_rc bundle contains %q = false, want true:\n%s", "File: input.txt", got)
	}
	if strings.Contains(got, "File: bundle.bundle") {
		t.Errorf("file_bundle -i all.file_bundle_rc bundle contains %q = true, want false:\n%s", "File: bundle.bundle", got)
	}
}

func TestTouchCreatesDefaultConfigAndReturns(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	args := []string{}
	setFlagArgs(t, args...)

	touch()

	configPath := "_" + defaultConfigExt
	got := readConfig(t, configPath)
	want := Config{
		Entry:   []string{defaultTouchEntryPattern()},
		Exclude: []string{".bundle", ".bundle.txt"},
		Output:  defaultName,
	}
	if got.Output != want.Output {
		t.Errorf("touch() with args %q config output = %q, want %q", args, got.Output, want.Output)
	}
	if got.Description != want.Description {
		t.Errorf("touch() with args %q config description = %q, want %q", args, got.Description, want.Description)
	}
	if !slices.Equal(got.Entry, want.Entry) {
		t.Errorf("touch() with args %q config entry = %q, want %q", args, got.Entry, want.Entry)
	}
	if !slices.Equal(got.Exclude, want.Exclude) {
		t.Errorf("touch() with args %q config exclude = %q, want %q", args, got.Exclude, want.Exclude)
	}
}

func TestTouchDirCreatesBundleConfigAndMakefileAndReturns(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	args := []string{"dir"}
	setFlagArgs(t, args...)

	touch()

	configPath := filepath.Join("bundle", "_all"+defaultConfigExt)
	got := readConfig(t, configPath)
	want := Config{
		Entry:   []string{defaultTouchEntryPattern()},
		Exclude: []string{".bundle", ".bundle.txt"},
	}
	if got.Output != want.Output {
		t.Errorf("touch() with args %q config output = %q, want %q", args, got.Output, want.Output)
	}
	if !slices.Equal(got.Entry, want.Entry) {
		t.Errorf("touch() with args %q config entry = %q, want %q", args, got.Entry, want.Entry)
	}
	if !slices.Equal(got.Exclude, want.Exclude) {
		t.Errorf("touch() with args %q config exclude = %q, want %q", args, got.Exclude, want.Exclude)
	}

	makefilePath := filepath.Join("bundle", "Makefile")
	makefileBytes, err := os.ReadFile(makefilePath)
	if err != nil {
		t.Fatalf("touch() with args %q Makefile read error = %v, want nil", args, err)
	}
	makefileContent := string(makefileBytes)
	for _, want := range []string{
		"FILE_BUNDLE_RCS :=",
		"%.bundle.txt: %.file_bundle_rc",
		"file_bundle -v -i $< -o $@",
	} {
		if !strings.Contains(makefileContent, want) {
			t.Errorf("touch() with args %q Makefile content contains %q = false, want true:\n%s", args, want, makefileContent)
		}
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

func setFlagArgs(t *testing.T, args ...string) {
	t.Helper()

	originalCommandLine := flag.CommandLine
	flag.CommandLine = flag.NewFlagSet(t.Name(), flag.ContinueOnError)
	if err := flag.CommandLine.Parse(args); err != nil {
		t.Fatalf("flag.Parse(%q) error = %v, want nil", args, err)
	}
	t.Cleanup(func() {
		flag.CommandLine = originalCommandLine
	})
}

func runMainForTest(t *testing.T, args ...string) {
	t.Helper()

	originalArgs := os.Args
	originalInput := input
	originalOutput := output
	originalShrink := shrink
	originalTouchCMD := touchCMD
	originalVerbose := verbose
	originalLineCount := lineCount
	originalCharCount := charCount
	originalFileCount := fileCount
	t.Cleanup(func() {
		os.Args = originalArgs
		input = originalInput
		output = originalOutput
		shrink = originalShrink
		touchCMD = originalTouchCMD
		verbose = originalVerbose
		lineCount = originalLineCount
		charCount = originalCharCount
		fileCount = originalFileCount
	})

	setFlagArgs(t)
	registerTestFlags()
	os.Args = append([]string{"file_bundle"}, args...)
	input = ""
	output = ""
	shrink = false
	touchCMD = false
	verbose = false
	lineCount = 0
	charCount = 0
	fileCount = 0

	main()
}

func runMainExpectingExit(t *testing.T, args ...string) int {
	t.Helper()

	exe := buildTestBinary(t)
	cmd := exec.Command(exe, args...)
	cmd.Dir = mustGetwd(t)

	err := cmd.Run()
	if err == nil {
		t.Fatal("file_bundle exited successfully, want failure")
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("file_bundle run error = %v, want exit error", err)
	}
	return exitErr.ExitCode()
}

func buildTestBinary(t *testing.T) string {
	t.Helper()

	exe := filepath.Join(t.TempDir(), "file_bundle")
	cmd := exec.Command("go", "build", "-o", exe, ".")
	cmd.Dir = sourceRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build error = %v, output:\n%s", err, output)
	}
	return exe
}

func sourceRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	return filepath.Dir(filename)
}

func registerTestFlags() {
	flag.StringVar(&input, "i", "", "input .file_bundle_rc file name(s)")
	flag.StringVar(&output, "o", "", "output file name")
	flag.BoolVar(&shrink, "s", false, "shrink mode: trim unnecessary white space")
	flag.BoolVar(&verbose, "v", false, "verbose mode")
	flag.BoolVar(&touchCMD, "touch", false, "initialize a default _.file_bundle_rc")
}

func writeConfig(t *testing.T, path string, config Config) {
	t.Helper()

	var buf strings.Builder
	if err := toml.NewEncoder(&buf).Encode(config); err != nil {
		t.Fatalf("toml.Encode(%q) error = %v, want nil", path, err)
	}
	if err := os.WriteFile(path, []byte(buf.String()), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", path, err)
	}
}

func readConfig(t *testing.T, path string) Config {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil", path, err)
	}
	var config Config
	if _, err := toml.Decode(string(content), &config); err != nil {
		t.Fatalf("toml.Decode(%q) error = %v, want nil", path, err)
	}
	return config
}

func defaultTouchEntryPattern() string {
	slash := string(os.PathSeparator)
	return "." + slash + "**" + slash + "*"
}
