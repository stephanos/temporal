package wasi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func helperCorpusClusterAssets(kase map[string]any, guestPath string) (map[string]any, error) {
	derived, _ := kase["derived"].(map[string]any)
	packet, _ := derived["cluster_packet"].(map[string]any)
	assets, _ := packet["assets"].(map[string]any)
	if len(assets) == 0 {
		return kase, nil
	}
	encoded, err := helperCorpusMarshal(kase)
	if err != nil {
		return nil, err
	}
	decoded, err := decodeCorpusJSON(encoded)
	if err != nil {
		return nil, err
	}
	out := decoded.(map[string]any)
	var replace func(any, string, string)
	replace = func(value any, key, replacement string) {
		switch value := value.(type) {
		case map[string]any:
			for field, item := range value {
				if text, ok := item.(string); ok && text == key {
					value[field] = replacement
				} else {
					replace(item, key, replacement)
				}
			}
		case []any:
			for index, item := range value {
				if text, ok := item.(string); ok && text == key {
					value[index] = replacement
				} else {
					replace(item, key, replacement)
				}
			}
		}
	}
	for name, asset := range assets {
		if name == "" || strings.IndexFunc(name, func(r rune) bool {
			return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_'
		}) >= 0 {
			return nil, fmt.Errorf("invalid cluster asset name %q", name)
		}
		wat, ok := asset.(string)
		if !ok {
			return nil, fmt.Errorf("cluster asset %s must be WAT text", name)
		}
		path := filepath.Join(filepath.Dir(guestPath), name+".wat")
		if err := os.WriteFile(path, []byte(wat), 0o600); err != nil {
			return nil, err
		}
		replace(out, "$cluster_asset."+name+".path", path)
	}
	return out, nil
}

func helperCorpusClusterIncarnationPrefix(fields map[string]any, instance string) (string, error) {
	raw, present := fields["incarnation"]
	if !present {
		return instance, nil
	}
	value, ok := raw.(json.Number)
	if !ok || instance == "" {
		return "", fmt.Errorf("invalid callback incarnation")
	}
	n, err := strconv.ParseUint(value.String(), 10, 64)
	if err != nil || n <= 1 {
		return "", fmt.Errorf("invalid restarted callback incarnation")
	}
	return instance + `"incarnation":` + strconv.FormatUint(n, 10) + ",", nil
}
