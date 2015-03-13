package core

import (
	"os"
	"path/filepath"
	"strings"
)

// GetDistPath returns basePath (where tmail binaries is)
func GetBasePath() string {
	p, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	return p
}

// Remove trailing and ending brackets (<string> -> string)
func RemoveBrackets(s string) string {
	if strings.HasPrefix(s, "<") {
		s = s[1:]
	}
	if strings.HasSuffix(s, ">") {
		s = s[0 : len(s)-1]
	}
	return s
}

// TODO: replace by sort package
// Check if a string is in a Slice of string
func IsStringInSlice(str string, s []string) (found bool) {
	found = false
	for _, t := range s {
		if t == str {
			found = true
			break
		}
	}
	return
}

// stripQuotes remove trailing and ending "
func StripQuotes(s string) string {
	if s == "" {
		return s
	}

	if s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// IsIpv4 return true if ip is ipV4
func IsIpV4(ip string) bool {
	if len(ip) > 15 {
		return false
	}
	return true
}
