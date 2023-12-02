package main

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed README.md
var help string

func printHelp() {
	fmt.Println("\n●", help)
}

func shrinkContent(rawContent []byte) string {
	if !shrink {
		return string(rawContent)
	}

	lines := strings.Split(string(rawContent), "\n")
	trimmedLines := make([]string, 0, len(lines))
	var lastLineWasEmpty bool

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == "" {
			if !lastLineWasEmpty {
				trimmedLines = append(trimmedLines, trimmedLine)
				lastLineWasEmpty = true
			}
		} else {
			trimmedLines = append(trimmedLines, trimmedLine)
			lastLineWasEmpty = false
		}
	}

	return strings.Join(trimmedLines, "\n")
}
