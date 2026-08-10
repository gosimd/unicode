package utf

import "unicode/utf8"

// Valid reports whether p consists entirely of valid UTF-8.
func Valid(p []byte) bool {
	return utf8.Valid(p)
}

// ValidString reports whether s consists entirely of valid UTF-8.
func ValidString(s string) bool {
	return utf8.ValidString(s)
}
