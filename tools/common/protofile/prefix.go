package protofile

import (
	"errors"
	"fmt"
	"path"
	"strings"
)

func NormalizePrefix(value string) (string, error) {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	if value == "" {
		return "", errors.New("prefix cannot be empty")
	}
	trailingSlash := strings.HasSuffix(value, "/")
	value = path.Clean(value)
	if value == "." || path.IsAbs(value) || value == ".." || strings.HasPrefix(value, "../") {
		return "", fmt.Errorf("prefix %q is unsafe", value)
	}
	if trailingSlash {
		value += "/"
	}
	return value, nil
}
