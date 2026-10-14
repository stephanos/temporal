package sqlitebusy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const Policy = "gomad-wasm.sqlite-busy-cooperative/v1"

type Dependency struct {
	Path, Version string
	Replaced      bool
}

type Adapter struct {
	Policy, OriginalSHA256, ReplacementSHA256 string
	Source                                    []byte
}

func Prepare(dependency Dependency, source []byte) (Adapter, error) {
	if dependency.Path != "github.com/ncruces/go-sqlite3" || dependency.Version != "v0.35.6" || dependency.Replaced {
		return Adapter{}, fmt.Errorf("SQLite cooperative adapter requires unreplaced github.com/ncruces/go-sqlite3 v0.35.6")
	}
	original := sha256.Sum256(source)
	originalSHA256 := hex.EncodeToString(original[:])
	if originalSHA256 != "85d9934324e945f1209c69eab06a81faa13f68ae157aee7a25f426736e1e3dd6" {
		return Adapter{}, fmt.Errorf("SQLite cooperative adapter source identity mismatch: %s", originalSHA256)
	}
	before := []byte("\t\t\ttime.Sleep(time.Duration(rand.Int63() & sleepIncrement))\n\t\t}")
	after := []byte("\t\t\ttime.Sleep(time.Duration(rand.Int63() & sleepIncrement))\n\t\t} else {\n\t\t\truntime.Gosched()\n\t\t}")
	if bytes.Count(source, before) != 1 {
		return Adapter{}, fmt.Errorf("SQLite cooperative adapter requires the pinned busy-timeout branch")
	}
	replacement := bytes.Replace(source, before, after, 1)
	replacementHash := sha256.Sum256(replacement)
	return Adapter{Policy: Policy, OriginalSHA256: originalSHA256, ReplacementSHA256: hex.EncodeToString(replacementHash[:]), Source: replacement}, nil
}
