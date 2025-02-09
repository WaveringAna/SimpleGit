//utils/file.go

package utils

import (
	"path/filepath"
	"regexp"
	"strings"
)

type Symbol struct {
	Name string
	Type string
	Icon string
	Line int
}

func IsBinaryFile(content []byte) bool {
	size := len(content)
	if size > 512 {
		size = 512
	}

	for i := 0; i < size; i++ {
		if content[i] == 0 || (content[i] < 32 && content[i] != '\n' && content[i] != '\r' && content[i] != '\t') {
			return true
		}
	}
	return false
}

func ParseSymbols(content []byte) []Symbol {
	var symbols []Symbol
	lines := strings.Split(string(content), "\n")

	// Common patterns across languages
	patterns := []struct {
		regex string
		typ   string
		icon  string
	}{
		// Functions - catches async, static, public, private, etc.
		{`^[\s]*(?:async\s+)?(?:static\s+)?(?:public\s+)?(?:private\s+)?(?:protected\s+)?(?:func|function)\s+(?:\([^)]*\)\s+)?(\w+)`, "function", "ƒ"},

		// Arrow functions with explicit name
		{`^[\s]*(?:export\s+)?(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?\(.*?\)\s*=>`, "function", "ƒ"},

		// Classes/types
		{`^[\s]*(?:export\s+)?(?:abstract\s+)?(?:class|type)\s+(\w+)`, "class", "◇"},

		// Interfaces
		{`^[\s]*(?:export\s+)?interface\s+(\w+)`, "interface", "⬡"},

		// Constants
		{`^[\s]*(?:export\s+)?(?:const|final)\s+(\w+)`, "constant", "□"},

		// Variables
		{`^[\s]*(?:export\s+)?(?:var|let|private|public|protected)\s+(\w+)`, "variable", "○"},
		{`^[\s]*(\w+(?:\s*,\s*\w+)*)\s*:=`, "variable", "○"},

		// Methods
		{`^[\s]*(?:async\s+)?(?:static\s+)?(?:public\s+)?(?:private\s+)?(?:protected\s+)?(?:def|method)\s+(\w+)`, "method", "⌘"},

		// YAML keys (top level)
		{`^(\w+):(?:\s|$)`, "property", "⚑"},

		// YAML anchors
		{`^[\s]*&(\w+)\b`, "anchor", "⚓"},

		// JSON properties (with quotes)
		{`^[\s]*"(\w+)"\s*:`, "property", "⚑"},

		// JSON/YAML nested objects
		{`^[\s]*"?(\w+)"?\s*:\s*{`, "object", "⬡"},

		// JSON/YAML arrays
		{`^[\s]*"?(\w+)"?\s*:\s*\[`, "array", "▤"},
	}

	for lineNum, line := range lines {
		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern.regex)
			if matches := re.FindStringSubmatch(line); matches != nil {
				if pattern.typ == "variable" && strings.Contains(matches[1], ",") {
					// Split multiple variable declarations
					vars := strings.Split(matches[1], ",")
					for _, v := range vars {
						symbols = append(symbols, Symbol{
							Name: strings.TrimSpace(v),
							Type: pattern.typ,
							Icon: pattern.icon,
							Line: lineNum + 1,
						})
					}
				} else {
					symbols = append(symbols, Symbol{
						Name: matches[1],
						Type: pattern.typ,
						Icon: pattern.icon,
						Line: lineNum + 1,
					})
				}
			}
		}
	}

	return symbols
}

func GetFileIcon(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".go":
		return "fa-brands fa-golang"
	case ".js":
		return "fa-brands fa-square-js"
	case ".ts", ".tsx":
		return "fa-brands fa-js"
	case ".py":
		return "fa-brands fa-python"
	case ".rs":
		return "fa-brands fa-rust"
	case ".html":
		return "fa-brands fa-html5"
	case ".css":
		return "fa-brands fa-css3-alt"
	case ".php":
		return "fa-brands fa-php"
	case ".java":
		return "fa-brands fa-java"
	case ".rb":
		return "fa-brands fa-ruby"
	case ".md", ".markdown":
		return "fa-brands fa-markdown"
	case ".docker", "Dockerfile":
		return "fa-brands fa-docker"
	case ".git", ".gitignore":
		return "fa-brands fa-git-alt"
	default:
		return "fa-regular fa-file"
	}
}
