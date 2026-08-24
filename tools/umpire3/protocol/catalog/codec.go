package catalog

import (
	"crypto/sha256"
	"fmt"
	"io"

	"go.temporal.io/server/tools/umpire3/protocol/internal/codec"
)

const DefaultDecodeLimit = codec.DefaultDecodeLimit

func decodeStrictJSON(reader io.Reader, limit int64, kind string, destination any) error {
	return codec.DecodeStrictJSON(reader, limit, kind, destination)
}

func validHash(value string) bool {
	return codec.ValidHash(value)
}

func digestBytes(value []byte) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256(value))
}
