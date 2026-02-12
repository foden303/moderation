package hash

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"

	"github.com/cespare/xxhash/v2"
	"github.com/spaolacci/murmur3"
)

// Hash returns the hash value of data.
func Hash(data []byte) uint64 {
	return murmur3.Sum64(data)
}

func HashTextSha256(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func FastHash(s string) []byte {
	h := xxhash.Sum64String(s)
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, h)
	return buf
}
