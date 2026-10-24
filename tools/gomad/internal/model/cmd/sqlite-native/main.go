// sqlite-native projects actual Rust storage evidence through the retained Go reference.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/model/strictjson"
	"go.temporal.io/server/tools/gomad/internal/model/wasi"
)

type bounds struct {
	States     uint64 `json:"states"`
	Operations uint64 `json:"operations"`
	Depth      uint64 `json:"depth"`
	Bytes      uint64 `json:"bytes"`
	Work       uint64 `json:"work"`
}
type request struct {
	PrecrashPath   string `json:"precrash_image_path"`
	PrecrashSHA256 string `json:"precrash_image_sha256"`
	ModelIdentity  string `json:"model_identity"`
	StatePath      string `json:"storage_state_path"`
	StateSHA256    string `json:"storage_state_sha256"`
	LogPath        string `json:"storage_log_path"`
	LogSHA256      string `json:"storage_log_sha256"`
	Limits         bounds `json:"limits"`
}

func read(path string, maximum int64) ([]byte, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("native oracle requires absolute owned evidence paths")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maximum {
		return nil, errors.New("native oracle input type/capacity")
	}
	stream, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = stream.Close() }()
	actual, err := stream.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(info, actual) {
		return nil, errors.New("native oracle input changed")
	}
	raw, err := io.ReadAll(io.LimitReader(stream, maximum+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > maximum {
		return nil, errors.New("native oracle input capacity")
	}
	return raw, nil
}

func authenticated(path, digest string, maximum int64) ([]byte, error) {
	expected, err := hex.DecodeString(digest)
	if err != nil || len(expected) != 32 || hex.EncodeToString(expected) != digest {
		return nil, errors.New("native oracle requires canonical SHA256")
	}
	raw, err := read(path, maximum)
	if err != nil {
		return nil, err
	}
	actual := sha256.Sum256(raw)
	if hex.EncodeToString(actual[:]) != digest {
		return nil, errors.New("native oracle payload changed")
	}
	return raw, nil
}

func project(path string, output io.Writer) error {
	raw, err := read(path, 64<<10)
	if err != nil {
		return err
	}
	var input request
	if err := strictjson.Decode(raw, &input); err != nil {
		return err
	}
	if input.Limits != (bounds{32, 8, 8, 4 << 20, 1 << 16}) {
		return errors.New("native oracle changed original SQLite enumeration bounds")
	}
	state, err := authenticated(input.StatePath, input.StateSHA256, 8<<20)
	if err != nil {
		return err
	}
	log, err := authenticated(input.LogPath, input.LogSHA256, 32<<20)
	if err != nil {
		return err
	}
	precrash, err := authenticated(input.PrecrashPath, input.PrecrashSHA256, 9<<20)
	if err != nil {
		return err
	}
	model, err := hex.DecodeString(strings.TrimPrefix(input.ModelIdentity, "sha256:"))
	if err != nil || len(model) != 32 || "sha256:"+hex.EncodeToString(model) != input.ModelIdentity {
		return errors.New("native oracle requires canonical model identity")
	}
	enumeration, context, err := wasi.ProjectSQLiteCrashCatalogue(state, log, precrash, input.ModelIdentity)
	if err != nil {
		return err
	}
	return json.NewEncoder(output).Encode(struct {
		Enumeration any `json:"enumeration"`
		Context     any `json:"context"`
	}{enumeration, context})
}

func main() {
	input := flag.String("input", "", "authenticated actual storage evidence request")
	flag.Parse()
	if flag.NArg() != 0 || *input == "" {
		fmt.Fprintln(os.Stderr, "--input required; positional arguments refused")
		os.Exit(2)
	}
	if err := project(*input, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
