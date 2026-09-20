//go:build !windows

package cmd

// userLocaleName has nothing to add off Windows: the language comes from the
// LC_ALL/LC_MESSAGES/LANG/LANGUAGE variables.
func userLocaleName() string { return "" }
