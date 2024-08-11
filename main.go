/*
This program reads the first '*.file_bundle_rc' file from the current directory
and creates 'bundle.bundle'.

The '.file_bundle_rc' file is in TOML format and has a structure as shown below:

entry = ["file1", "file2", ...]

In the above structure, 'file1', 'file2', etc., are the paths of the files
that you want to bundle. The paths should be relative to the current directory.

When this program runs, it will read each file in the 'entry' list,
and append the contents of these files into a single 'bundle.bundle' file.
Each file appended to 'bundle.bundle' file will be preceded by a separator
line and the original path of the file.
*/
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar"
)

var (
	lineCount, charCount, fileCount int
	startTime                       = time.Now()
)

func main() {
	config := initConf()

	if err := validateOutputWithinWorkingDirectory(config.Output); err != nil {
		fmt.Printf("Refusing to write bundle output: %v\n", err)
		os.Exit(1)
	}

	outFile, err := os.Create(config.Output)
	if err != nil {
		fmt.Printf("Failed to create bundle file '%s'\nerr= %v\n", config.Output, err)
		os.Exit(1)
	}
	defer outFile.Close()

	// A map to store the paths of files, used to avoid duplicates.
	visited := make(map[string]bool)
	excluded := make(map[string]bool)

	// Populate the excluded map
	for _, pattern := range config.Exclude {
		matches, err := doublestar.Glob(pattern)
		if err != nil {
			fmt.Printf("Invalid pattern in Exclude: %s\n", pattern)
			continue
		}
		for _, match := range matches {
			excluded[match] = true
		}
	}

	// Process each entry.
	for _, pattern := range config.Entry {
		matches, err := doublestar.Glob(pattern)
		if err != nil {
			fmt.Printf("Invalid pattern in Entry: %s\n", pattern)
			continue
		}

		for _, match := range matches {
			if !visited[match] && !excluded[match] && isWithinWorkingDirectory(match) && !isOutputPath(match, config.Output) {
				info, err := os.Stat(match)
				if err != nil {
					fmt.Printf("Error accessing the path %s: %v", match, err)
					continue
				}
				if !info.IsDir() {
					processFile(match, outFile, config)
					visited[match] = true
				}
			}
		}
	}

	// 在程序结束时，输出报告
	fmt.Printf("=== Bundle (%s) created successfully\n", config.Output)
	if verbose {
		fmt.Printf(" - Execution Time: %v\n", time.Since(startTime))
		fmt.Printf(" - Total Files: %d\n", fileCount)
		fmt.Printf(" - Total Lines: %d\n", lineCount)
		fmt.Printf(" - Total Characters: %d\n", charCount)
		fmt.Println("==================================")
	}
}

func isOutputPath(path string, output string) bool {
	path = strings.TrimSpace(path)
	output = strings.TrimSpace(output)
	if path == "" || output == "" {
		return false
	}
	pathAbs, pathErr := filepath.Abs(path)
	outputAbs, outputErr := filepath.Abs(output)
	if pathErr != nil || outputErr != nil {
		return filepath.Clean(path) == filepath.Clean(output)
	}
	return filepath.Clean(pathAbs) == filepath.Clean(outputAbs)
}

func isOutputWithinWorkingDirectory(path string) bool {
	return validateOutputWithinWorkingDirectory(path) == nil
}

func validateOutputWithinWorkingDirectory(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("empty output path")
	}

	wd, wdErr := os.Getwd()
	pathAbs, pathErr := filepath.Abs(path)
	if wdErr != nil || pathErr != nil {
		return fmt.Errorf("resolve output path %q: %v", path, firstErr(wdErr, pathErr))
	}
	wd, wdErr = filepath.EvalSymlinks(wd)
	if wdErr != nil {
		return fmt.Errorf("resolve working directory: %w", wdErr)
	}

	if _, statErr := os.Lstat(pathAbs); statErr == nil {
		pathAbs, pathErr = filepath.EvalSymlinks(pathAbs)
		if pathErr != nil {
			return fmt.Errorf("resolve output path %q: %w", path, pathErr)
		}
	} else if os.IsNotExist(statErr) {
		parent, parentErr := filepath.EvalSymlinks(filepath.Dir(pathAbs))
		if parentErr != nil {
			return fmt.Errorf("output parent directory does not exist or cannot be resolved: %s", filepath.Dir(path))
		}
		pathAbs = filepath.Join(parent, filepath.Base(pathAbs))
	} else {
		return fmt.Errorf("inspect output path %q: %w", path, statErr)
	}

	if !isPathWithinRoot(wd, pathAbs) {
		return fmt.Errorf("output path is outside the working directory: %s", path)
	}
	return nil
}

func firstErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func isWithinWorkingDirectory(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}

	wd, wdErr := os.Getwd()
	pathAbs, pathErr := filepath.Abs(path)
	if wdErr != nil || pathErr != nil {
		return false
	}
	wd, wdErr = filepath.EvalSymlinks(wd)
	pathAbs, pathErr = filepath.EvalSymlinks(pathAbs)
	if wdErr != nil || pathErr != nil {
		return false
	}

	return isPathWithinRoot(wd, pathAbs)
}

func isPathWithinRoot(root string, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}

func processFile(path string, outFile *os.File, config Config) {
	rawContent, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		fmt.Printf("Could not read file '%s': %v", path, err)
		os.Exit(1)
	}

	fileContent := shrinkContent(rawContent)
	lineCount += len(strings.Split(fileContent, "\n"))
	charCount += len(fileContent)
	fileCount++

	timestamp := time.Now()
	_, _ = outFile.WriteString("==========\n")
	if config.Description != "" {
		_, _ = outFile.WriteString(fmt.Sprintf("!! %s\n", config.Description))
	}
	_, _ = outFile.WriteString(fmt.Sprintf("File: %s\n", path))
	_, _ = outFile.WriteString(fmt.Sprintf("Time: %s\n", timestamp.Format("2006-01-02 15:04:05")))
	_, _ = outFile.WriteString("==========\n")
	_, _ = outFile.WriteString(fileContent)
	_, _ = outFile.WriteString("\n")

	if verbose {
		fmt.Printf("Visited: %s\n", path)
	}
}
