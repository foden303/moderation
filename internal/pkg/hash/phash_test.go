package hash

import (
	"image"
	"image/color"
	"testing"
)

// createTestImage creates a simple test image.
func createTestImage(width, height int, fill color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, fill)
		}
	}
	return img
}

// createGradientImage creates a gradient test image.
func createGradientImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gray := uint8((x + y) * 255 / (width + height))
			img.Set(x, y, color.RGBA{gray, gray, gray, 255})
		}
	}
	return img
}

func TestPerceptualHasher_ComputePHash(t *testing.T) {
	ph := NewPerceptualHasher()
	img := createGradientImage(100, 100)

	hash, err := ph.ComputePHash(img)
	if err != nil {
		t.Fatalf("ComputePHash failed: %v", err)
	}

	if hash.Hash == 0 {
		t.Error("Expected non-zero hash")
	}
	if hash.HashType != PHash {
		t.Errorf("Expected PHash type, got %d", hash.HashType)
	}
	if hash.Width != 100 || hash.Height != 100 {
		t.Errorf("Expected 100x100, got %dx%d", hash.Width, hash.Height)
	}
}

func TestPerceptualHasher_ComputeAHash(t *testing.T) {
	ph := NewPerceptualHasher()
	img := createGradientImage(100, 100)

	hash, err := ph.ComputeAHash(img)
	if err != nil {
		t.Fatalf("ComputeAHash failed: %v", err)
	}

	if hash.HashType != AHash {
		t.Errorf("Expected AHash type, got %d", hash.HashType)
	}
}

func TestPerceptualHasher_ComputeDHash(t *testing.T) {
	ph := NewPerceptualHasher()
	img := createGradientImage(100, 100)

	hash, err := ph.ComputeDHash(img)
	if err != nil {
		t.Fatalf("ComputeDHash failed: %v", err)
	}

	if hash.HashType != DHash {
		t.Errorf("Expected DHash type, got %d", hash.HashType)
	}
}

func TestHammingDistance(t *testing.T) {
	tests := []struct {
		name     string
		hash1    uint64
		hash2    uint64
		expected int
	}{
		{
			name:     "identical",
			hash1:    0xFFFFFFFFFFFFFFFF,
			hash2:    0xFFFFFFFFFFFFFFFF,
			expected: 0,
		},
		{
			name:     "one bit different",
			hash1:    0xFFFFFFFFFFFFFFFE,
			hash2:    0xFFFFFFFFFFFFFFFF,
			expected: 1,
		},
		{
			name:     "completely different",
			hash1:    0x0000000000000000,
			hash2:    0xFFFFFFFFFFFFFFFF,
			expected: 64,
		},
		{
			name:     "half different",
			hash1:    0x00000000FFFFFFFF,
			hash2:    0xFFFFFFFF00000000,
			expected: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HammingDistance(tt.hash1, tt.hash2)
			if result != tt.expected {
				t.Errorf("HammingDistance(%x, %x) = %d; want %d", tt.hash1, tt.hash2, result, tt.expected)
			}
		})
	}
}

func TestIsSimilar(t *testing.T) {
	h1 := &ImageHash{Hash: 0xFFFFFFFFFFFFFFFF}
	h2 := &ImageHash{Hash: 0xFFFFFFFFFFFFFFF0} // 4 bits different

	if !IsSimilar(h1, h2, 5) {
		t.Error("Expected images to be similar with threshold 5")
	}
	if IsSimilar(h1, h2, 3) {
		t.Error("Expected images to NOT be similar with threshold 3")
	}
}

func TestSimilarityScore(t *testing.T) {
	h1 := &ImageHash{Hash: 0xFFFFFFFFFFFFFFFF}
	h2 := &ImageHash{Hash: 0xFFFFFFFFFFFFFFFF}

	score := SimilarityScore(h1, h2)
	if score != 100.0 {
		t.Errorf("Expected similarity score 100, got %f", score)
	}

	h3 := &ImageHash{Hash: 0x0000000000000000}
	score = SimilarityScore(h1, h3)
	if score != 0.0 {
		t.Errorf("Expected similarity score 0, got %f", score)
	}
}

func TestImageHash_String(t *testing.T) {
	h := &ImageHash{Hash: 0xDEADBEEF12345678}
	expected := "deadbeef12345678"
	if h.String() != expected {
		t.Errorf("String() = %s; want %s", h.String(), expected)
	}
}

func TestSameImageIdenticalHash(t *testing.T) {
	ph := NewPerceptualHasher()
	img := createGradientImage(100, 100)

	hash1, _ := ph.ComputePHash(img)
	hash2, _ := ph.ComputePHash(img)

	if hash1.Hash != hash2.Hash {
		t.Error("Same image should produce identical hash")
	}
}

func TestDifferentImagesProduceDifferentHashes(t *testing.T) {
	ph := NewPerceptualHasher()

	white := createTestImage(100, 100, color.White)
	black := createTestImage(100, 100, color.Black)

	h1, _ := ph.ComputePHash(white)
	h2, _ := ph.ComputePHash(black)

	if h1.Hash == h2.Hash {
		t.Error("Different images should produce different hashes")
	}
}

func BenchmarkComputePHash(b *testing.B) {
	ph := NewPerceptualHasher()
	img := createGradientImage(500, 500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ph.ComputePHash(img)
	}
}

func BenchmarkHammingDistance(b *testing.B) {
	h1 := uint64(0xDEADBEEF12345678)
	h2 := uint64(0xCAFEBABE87654321)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		HammingDistance(h1, h2)
	}
}
