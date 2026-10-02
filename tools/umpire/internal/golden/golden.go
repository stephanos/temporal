// Package golden supports migration tests; production packages must not import it.
package golden

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

//go:embed config.json
var configBytes []byte

type Substitution struct{ Old, New string }
type Config struct {
	Inventory []string       `json:"ir_inventory"`
	Paths     []Substitution `json:"source_path_substitutions"`
	Labels    []Substitution `json:"source_label_substitutions"`
}

func Configuration() (Config, error) {
	var c Config
	err := json.Unmarshal(configBytes, &c)
	return c, err
}

func Root() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("repository go.mod not found")
		}
		dir = parent
	}
}

func (c Config) Inputs(root string) (map[string]*umpirespb.Model, error) {
	base := "model/scalav2"
	if _, err := os.Stat(filepath.Join(root, base)); errors.Is(err, fs.ErrNotExist) {
		base = "model"
	} else if err != nil {
		return nil, err
	}
	expected := make(map[string]string, len(c.Inventory))
	for _, path := range c.Inventory {
		expected[base+strings.TrimPrefix(path, "model/scalav2")] = path
	}
	found := map[string]bool{}
	for _, dir := range []string{"ir", "lifter/testdata/lifts/expected"} {
		paths, err := filepath.Glob(filepath.Join(root, base, dir, "*.json"))
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return nil, err
			}
			if _, ok := expected[rel]; !ok {
				return nil, fmt.Errorf("unknown IR inventory entry %s", rel)
			}
			found[rel] = true
		}
	}
	out := map[string]*umpirespb.Model{}
	for _, path := range slices.Sorted(maps.Keys(expected)) {
		if !found[path] {
			return nil, fmt.Errorf("missing IR inventory entry %s", path)
		}
		encoded, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			return nil, err
		}
		m := new(umpirespb.Model)
		if err := protojson.Unmarshal(encoded, m); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		out[expected[path]] = m
	}
	return out, nil
}

func (c Config) Migrate(original *umpirespb.Model) (*umpirespb.Model, error) {
	m := proto.CloneOf(original)
	var err error
	m.Source, err = substitute(m.GetSource(), c.Labels)
	if err != nil {
		return nil, err
	}
	err = positions(m.ProtoReflect(), func(p protoreflect.Message) error {
		field := p.Descriptor().Fields().ByName("file")
		mapped, err := substitute(p.Get(field).String(), c.Paths)
		if err != nil {
			return err
		}
		p.Set(field, protoreflect.ValueOfString(mapped))
		return nil
	})
	return m, err
}

func substitute(value string, substitutions []Substitution) (string, error) {
	for _, s := range substitutions {
		if value == s.Old {
			return s.New, nil
		}
	}
	return "", fmt.Errorf("unlisted source %q", value)
}

func positions(m protoreflect.Message, visit func(protoreflect.Message) error) error {
	position := (&umpirespb.Position{}).ProtoReflect().Descriptor().FullName()
	var result error
	m.Range(func(f protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if f.Message() == nil || f.IsMap() {
			return true
		}
		walk := func(child protoreflect.Message) error {
			if child.Descriptor().FullName() == position {
				return visit(child)
			}
			return positions(child, visit)
		}
		if f.IsList() {
			for i := range v.List().Len() {
				if result = walk(v.List().Get(i).Message()); result != nil {
					return false
				}
			}
		} else {
			result = walk(v.Message())
		}
		return result == nil
	})
	return result
}

func (c Config) Match(original, current *umpirespb.Model) (bool, error) {
	if proto.Equal(original, current) {
		return false, nil
	}
	mapped, err := c.Migrate(original)
	if err != nil {
		return false, err
	}
	if !proto.Equal(mapped, current) {
		return false, errors.New("IR differs outside the closed source migration")
	}
	return true, nil
}

func JSON(v any) ([]byte, error) {
	encoded, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

func Proto(m proto.Message) ([]byte, error) {
	encoded, err := protojson.Marshal(m)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := json.Compact(&out, encoded); err != nil {
		return nil, err
	}
	return append(out.Bytes(), '\n'), nil
}

func Digest(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }

func Capture(dir string, files map[string][]byte) error {
	if err := os.Mkdir(dir, 0755); err != nil {
		return err
	}
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if !filepath.IsLocal(name) {
			return fmt.Errorf("unsafe capture path %q", name)
		}
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		file, err := os.OpenFile(path+".gz", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return err
		}
		compressed := gzip.NewWriter(file)
		_, writeErr := compressed.Write(files[name])
		if err := errors.Join(writeErr, compressed.Close(), file.Close()); err != nil {
			return err
		}
	}
	return nil
}

func Read(dir string) (map[string][]byte, error) {
	out := map[string][]byte{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(path, ".gz") {
			return fmt.Errorf("unexpected baseline file %s", path)
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		reader, err := gzip.NewReader(file)
		if err != nil {
			return errors.Join(err, file.Close())
		}
		data, readErr := io.ReadAll(reader)
		if err := errors.Join(readErr, reader.Close(), file.Close()); err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		out[strings.TrimSuffix(rel, ".gz")] = data
		return nil
	})
	return out, err
}

func Compare(expected, actual map[string][]byte) error {
	for _, key := range slices.Sorted(maps.Keys(expected)) {
		got, exists := actual[key]
		if !exists {
			return fmt.Errorf("missing golden entry %s", key)
		}
		if !bytes.Equal(expected[key], got) {
			return fmt.Errorf("golden difference %s: expected sha256 %s, got %s", key, Digest(expected[key]), Digest(got))
		}
	}
	for _, key := range slices.Sorted(maps.Keys(actual)) {
		if _, exists := expected[key]; !exists {
			return fmt.Errorf("unknown golden entry %s", key)
		}
	}
	return nil
}
