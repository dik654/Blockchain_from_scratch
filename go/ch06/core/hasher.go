package core

import (
	"crypto/sha256"

	"github.com/dik654/Blockchain_from_scratch/go/ch05/types"
)

type Hasher[T any] interface {
	Hash(T) types.Hash
}

type BlockHasher struct{}

func (BlockHasher) Hash(b *Block) types.Hash {
	h := sha256.Sum256(b.HeaderData())
	return types.Hash(h)
}
