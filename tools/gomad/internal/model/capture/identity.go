package captured

import (
	"crypto/sha256"
	"encoding/hex"
)

type SHA256 string

func HashBytes(data []byte) SHA256 {
	digest := sha256.Sum256(data)
	return SHA256("sha256:" + hex.EncodeToString(digest[:]))
}
