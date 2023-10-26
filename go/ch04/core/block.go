package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"

	"github.com/dik654/Blockchain_from_scratch/go/ch04/crypto"
	"github.com/dik654/Blockchain_from_scratch/go/ch04/types"
)

type Header struct {
	Version uint32
	// *
	// 트랜잭션 데이터 해시
	DataHash      types.Hash
	PrevBlockHash types.Hash
	Timestamp     int64
	Height        uint32
	Nonce         uint64
}

type Block struct {
	// *
	// 헤더 구조체의 복사본이 아닌 실제 Header 데이터의 위치만 저장하여
	// 헤더와 블록을 연결
	*Header
	Transactions []Transaction
	// *
	// 이 블록을 validate한 유저의 공개키
	Validator crypto.PublicKey
	// *
	// 그 유저의 서명
	Signature *crypto.Signature
	hash      types.Hash
}

// *
// 새 블록 생성
// 및 포인터 리턴
func NewBlock(h *Header, txx []Transaction) *Block {
	return &Block{
		Header:       h,
		Transactions: txx,
	}
}

// *
func (b *Block) Sign(privKey crypto.PrivateKey) error {
	// 개인 키로 직렬화된 헤더 데이터 서명
	sig, err := privKey.Sign(b.HeaderData())
	if err != nil {
		return err
	}

	// 서명한 개인키의 공개키 Validator로 설정
	b.Validator = privKey.PublicKey()
	// 생성한 서명 데이터의 포인터 Signatured에 지정
	b.Signature = sig

	return nil
}

// *
func (b *Block) Verify() error {
	if b.Signature == nil {
		return fmt.Errorf("block has no signature")
	}

	if !b.Signature.Verify(b.Validator, b.HeaderData()) {
		return fmt.Errorf("block has invalid signature")
	}

	return nil
}

// *
func (b *Block) Encode(w io.Writer, enc Encoder[*Block]) error {
	// 인코딩한 블록
	return enc.Encode(w, b)
}

// *
func (b *Block) Decode(r io.Reader, dec Decoder[*Block]) error {
	return dec.Decode(r, b)
}

// *
// 해시 생성 함수
func (b *Block) Hash(hasher Hasher[*Block]) types.Hash {
	// 블록에 해시 생성이 안된 상태면 새로 만들어 넣어서
	if b.hash.IsZero() {
		b.hash = hasher.Hash(b)
	}

	// 블록의 해시를 리턴
	return b.hash
}

// *
func (b *Block) HeaderData() []byte {
	// 버퍼 생성
	buf := &bytes.Buffer{}
	// 이진 직렬화용 gob 인코더 생성
	enc := gob.NewEncoder(buf)
	// 블록의 헤더 인코딩
	enc.Encode(b.Header)

	// 직렬화한 bytes 슬라이스 리턴
	return buf.Bytes()
}
