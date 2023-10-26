package core

import (
	"crypto/sha256"

	"github.com/dik654/Blockchain_from_scratch/go/ch04/types"
)

// *
type Hasher[T any] interface {
	Hash(T) types.Hash
}

// *
type BlockHasher struct{}

// *
func (BlockHasher) Hash(b *Block) types.Hash {
	// 헤더의 bytes 슬라이스 해시화
	h := sha256.Sum256(b.HeaderData())
	// 해시타입으로 변환하여 리턴
	return types.Hash(h)
}
