package core

import (
	"io"

	"github.com/dik654/Blockchain_from_scratch/go/ch04/crypto"
)

type Transaction struct {
	// 트랜잭션 데이터 배열
	Data []byte
	// *
	PublicKey crypto.PublicKey
	Signature *crypto.Signature
}

// 구현 예정
func (tx *Transaction) EncodeBinary(w io.Writer) error { return nil }

func (tx *Transaction) DecodeBinary(r io.Reader) error { return nil }
