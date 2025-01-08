//utils/file.go

package utils

import (
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
		{`^[\s]*(?:async\s+)?(?:static\s+)?(?:public\s+)?(?:private\s+)?(?:protected\s+)?(?:function|func)\s+(\w+)`, "function", "ƒ"},

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
				symbols = append(symbols, Symbol{
					Name: matches[1],
					Type: pattern.typ,
					Icon: pattern.icon,
					Line: lineNum + 1,
				})
			}
		}
	}

	return symbols
}
