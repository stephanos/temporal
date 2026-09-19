package cli

import "strings"

func quoteArgument(value string) string {
	if value != "" && strings.IndexFunc(value, func(character rune) bool {
		return character <= ' ' || strings.ContainsRune("'\"\\$`;&|<>(){}[]*?!", character)
	}) == -1 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
