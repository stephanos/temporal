package world

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
)

func choiceRank(seed Seed, equivalenceClass string, eventID EventID) [sha256.Size]byte {
	seedInput := append([]byte("gomad3/world/seed/v1\x00"), uint64Bytes(uint64(seed))...)
	key := sha256.Sum256(seedInput)
	hasher := hmac.New(sha256.New, key[:])
	_, _ = hasher.Write([]byte("gomad3/world/equivalent-event-order/v1\x00"))
	_, _ = hasher.Write(uint64Bytes(uint64(len(equivalenceClass))))
	_, _ = hasher.Write([]byte(equivalenceClass))
	_, _ = hasher.Write(uint64Bytes(uint64(eventID)))
	var rank [sha256.Size]byte
	copy(rank[:], hasher.Sum(nil))
	return rank
}

func uint64Bytes(value uint64) []byte {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	return encoded[:]
}
