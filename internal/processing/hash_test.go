package processing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContentHash_KnownEmpty(t *testing.T) {
	hash := ContentHash([]byte{})
	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", hash)
}

func TestContentHash_Deterministic(t *testing.T) {
	data := []byte("hello world")
	h1 := ContentHash(data)
	h2 := ContentHash(data)
	assert.Equal(t, h1, h2)
}

func TestContentHash_DifferentInputs(t *testing.T) {
	h1 := ContentHash([]byte("a"))
	h2 := ContentHash([]byte("b"))
	assert.NotEqual(t, h1, h2)
}

func TestContentHash_Length(t *testing.T) {
	inputs := [][]byte{
		{},
		[]byte("short"),
		make([]byte, 10000),
	}
	for _, data := range inputs {
		hash := ContentHash(data)
		assert.Len(t, hash, 64)
	}
}

func TestHashPrefix_Normal(t *testing.T) {
	hash := "e3b0c44298fc1c149afbf4c8996fb924"
	assert.Equal(t, "e3b0c44298fc", HashPrefix(hash, 12))
}

func TestHashPrefix_LongerThanHash(t *testing.T) {
	hash := "abcd"
	assert.Equal(t, "abcd", HashPrefix(hash, 100))
}

func TestHashPrefix_Zero(t *testing.T) {
	hash := "abcdef"
	assert.Equal(t, "abcdef", HashPrefix(hash, 0))
}
