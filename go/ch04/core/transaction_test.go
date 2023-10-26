package core

import (
	"testing"

	"github.com/dik654/Blockchain_from_scratch/go/ch04/crypto"
	"github.com/stretchr/testify/assert"
)

// *
func TestSignTransaction(t *testing.T) {
	// 개인 키 생성
	privKey := crypto.GeneratePrivateKey()
	// foo라는 트랜잭션이 들어있는 트랜잭션 구조체 생성
	tx := &Transaction{
		Data: []byte("foo"),
	}

	// 생성한 개인키로 트랜잭션 서명
	assert.Nil(t, tx.Sign(privKey))
	// 서명이 생성되었는지 체크
	assert.NotNil(t, tx.Signature)
}

// *
func TestVerifyTransaction(t *testing.T) {
	// 개인키 생성
	privKey := crypto.GeneratePrivateKey()
	// 트랜잭션 생성
	tx := &Transaction{
		Data: []byte("foo"),
	}

	// 개인키로 트랜잭션 서명
	assert.Nil(t, tx.Sign(privKey))
	// 트랜잭션 검증
	assert.Nil(t, tx.Verify())

	otherPrivKey := crypto.GeneratePrivateKey()
	tx.PublicKey = otherPrivKey.PublicKey()

	assert.NotNil(t, tx.Verify())
}
