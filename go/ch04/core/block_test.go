package core

import (
	"fmt"
	"testing"
	"time"

	"github.com/dik654/Blockchain_from_scratch/go/ch04/crypto"
	"github.com/dik654/Blockchain_from_scratch/go/ch04/types"
	"github.com/stretchr/testify/assert"
)

// *
// 헬퍼 함수
func randomBlock(height uint32) *Block {
	header := &Header{
		Version:       1,
		PrevBlockHash: types.RandomHash(),
		Height:        height,
		Timestamp:     time.Now().UnixNano(),
	}
	tx := Transaction{
		Data: []byte("foo"),
	}
	return NewBlock(header, []Transaction{tx})
}

// *
func TestHashBlock(t *testing.T) {
	// 헬퍼함수로 임의의 genesis 블록을 생성한 뒤
	b := randomBlock(0)
	// 블록을 해시화하여 테스트 콘솔에 뿌리기
	fmt.Println(b.Hash(BlockHasher{}))
}

// *
// 블록 서명 과정 테스트
func TestSignBlock(t *testing.T) {
	// 개인키 생성
	privKey := crypto.GeneratePrivateKey()
	// genesis 블록 생성
	b := randomBlock(0)
	// 개인키로 블록 서명하고 실제로 생성이 됐는지 체크
	assert.Nil(t, b.Sign(privKey))
	// 생성된 서명이 블록의 Signature에 저장됐는지
	assert.NotNil(t, b.Signature)
}

// *
// 서명 증명 테스트
func TestVerifyBlock(t *testing.T) {
	// A 개인키 생성
	privKey := crypto.GeneratePrivateKey()
	// genesis 블록 생성
	b := randomBlock(0)

	// A 개인키로 서명이 되지는지 체크
	assert.Nil(t, b.Sign(privKey))
	// 저장된 서명이 올바르게 verify가 되는지 체크
	assert.Nil(t, b.Verify())

	// B 개인키 생성
	otherPrivKey := crypto.GeneratePrivateKey()
	// Validator로 새로 생성한 B 개인키의 공개키 등록
	b.Validator = otherPrivKey.PublicKey()

	// A의 개인키로 만든 서명을 B 개인키로 verify하려 했을 때
	// 의도대로 실패하는지 체크
	assert.NotNil(t, b.Verify())
}
