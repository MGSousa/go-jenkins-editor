package main

import (
	"bytes"
	"fmt"
	"html"
	"strings"
)

var (
	firstStr = "<script>"
	lastStr = "</script>"
)

// ConcatBytes
func ConcatBytes(b []byte, middle string) string {
	fResult := bytes.SplitAfter(b, []byte(firstStr))
	lResult := bytes.SplitAfter(b, []byte(lastStr))
	return fmt.Sprintf("%s%s%s%s", fResult[0], middle, lastStr, lResult[1])
}

// Normalize
func Normalize(content string, escapeHtml bool) string {
	content = strings.TrimPrefix(strings.TrimSpace(content), "\"")
	content = strings.TrimSuffix(content, "\"")

	content = strings.ReplaceAll(content, "\\n", "\n")
	content = strings.ReplaceAll(content, "\\\n", "\\n")
	content = strings.ReplaceAll(content, "\\t", "\t")
	content = strings.ReplaceAll(content, "\\\"", "\"")
	content = strings.ReplaceAll(content, "\\\\", "\\")
	if escapeHtml {
		content = html.EscapeString(content)
	}

	return strings.ReplaceAll(content, "\\\\$", "\\$")
}
