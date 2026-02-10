package hash

import (
	"testing"
)

const (
	text = "foden_ngo"
)

func BenchmarkMurmur3(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Hash([]byte(text))
	}
}
