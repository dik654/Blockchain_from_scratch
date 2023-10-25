package core

import "io"

type Transaction struct {
	// 트랜잭션 데이터 배열
	Data []byte
}

// 구현 예정
func (tx *Transaction) EncodeBinary(w io.Writer) error { return nil }

func (tx *Transaction) DecodeBinary(r io.Reader) error { return nil }
