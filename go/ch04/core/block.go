package core

import (
	"github.com/dik654/Blockchain_from_scratch/go/ch04/crypto"
	"github.com/dik654/Blockchain_from_scratch/go/ch04/types"
)

type Header struct {
	Version uint32
	// *
	// 트랜잭션 데이터 해시
	DataHash  types.hash
	PrevBlock types.Hash
	Timestamp int64
	Height    uint32
	Nonce     uint64
}

type Block struct {
	Header
	Transactions []Transaction
	// *
	Validator crypto.PublicKey
	// *
	Signature *crypto.Signature
	hash      types.Hash
}
