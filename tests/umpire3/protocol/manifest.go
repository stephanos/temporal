package protocol

import (
	"encoding/json"
	"io"
)

const FormatVersion = "umpire3/v2"

type Toolchain struct {
	Lean string `json:"lean"`
}

type Manifest struct {
	FormatVersion string    `json:"formatVersion"`
	Toolchain     Toolchain `json:"toolchain"`
}

func NewEmptyManifest(leanVersion string) Manifest {
	return Manifest{
		FormatVersion: FormatVersion,
		Toolchain: Toolchain{
			Lean: leanVersion,
		},
	}
}

func WriteManifest(writer io.Writer, manifest Manifest) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(manifest)
}
