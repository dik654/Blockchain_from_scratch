package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"io"

	"github.com/dik654/Blockchain_from_scratch/go/ch03/types"
)

type Header struct {
	// 헤더끼리만 해싱하여 블록으로 생성할 수 있도록 트랜잭션 정보는 들어가지 않는다
	Version uint32
	// 이전 블록 해시
	PrevBlock types.Hash
	// 유닉스 timestamp는 uint가 없던 시절에 만들어져 int64로 되어있다
	Timestamp int64
	// 블록 높이
	Height uint32
	// 블록 난이도
	Nonce uint64
}

// Version 정보부터 차례대로 writer에 bytes로 쓰기
func (h *Header) EncodeBinary(w io.Writer) error {
	if err := binary.Write(w, binary.LittleEndian, &h.Version); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, &h.PrevBlock); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, &h.Timestamp); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, &h.Height); err != nil {
		return err
	}
	return binary.Write(w, binary.LittleEndian, &h.Nonce)
}

// reader로 bytes를 읽어서 Version부터 순서대로 Header의 변수에 넣기
func (h *Header) DecodeBinary(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &h.Version); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &h.PrevBlock); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &h.Timestamp); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &h.Height); err != nil {
		return err
	}
	return binary.Read(r, binary.LittleEndian, &h.Nonce)
}

type Block struct {
	Header
	Transactions []Transaction

	hash types.Hash
}

func (b *Block) Hash() types.Hash {
	buf := &bytes.Buffer{}
	// 블록 헤더를 bytes로 인코딩하여 버퍼에 저장
	b.Header.EncodeBinary(buf)

	// 초기화된 적이 없을 때만 해싱
	if b.hash.IsZero() {
		// 인코딩된 블록헤더를 sha256으로 해싱하여 블록 객체의 hash에 저장
		b.hash = types.Hash(sha256.Sum256(buf.Bytes()))
	}

	return b.hash
}

// 헤더와 마찬가지로 인, 디코딩
func (b *Block) EncodeBinary(w io.Writer) error {
	if err := b.Header.EncodeBinary(w); err != nil {
		return err
	}

	for _, tx := range b.Transactions {
		if err := tx.EncodeBinary(w); err != nil {
			return err
		}
	}

	return nil
}

func (b *Block) DecodeBinary(r io.Reader) error {
	if err := b.Header.DecodeBinary(r); err != nil {
		return err
	}

	for _, tx := range b.Transactions {
		if err := tx.DecodeBinary(r); err != nil {
			return err
		}
	}
	return nil
}
