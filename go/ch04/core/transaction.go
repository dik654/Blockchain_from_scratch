package core

import (
	"fmt"

	"github.com/dik654/Blockchain_from_scratch/go/ch04/crypto"
)

type Transaction struct {
	// 트랜잭션 데이터 배열
	Data []byte
	// *
	PublicKey crypto.PublicKey
	Signature *crypto.Signature
}

// *
func (tx *Transaction) Sign(privKey crypto.PrivateKey) error {
	// 개인키로 서명 생성
	// r, s값을 갖는 Signature 구조체 포인터 리턴
	sig, err := privKey.Sign(tx.Data)
	if err != nil {
		return err
	}

	// 트랜잭션 내부에 서명한 개인키의 공개키
	tx.PublicKey = privKey.PublicKey()
	// 및 서명 구조체 포인터 저장
	tx.Signature = sig

	return nil
}

// *
func (tx *Transaction) Verify() error {
	// 만약 서명이 없다면 에러 발생
	if tx.Signature == nil {
		return fmt.Errorf("transaction has no signature")
	}

	// 만약 서명 검증이 실패했다면 에러 발생
	if !tx.Signature.Verify(tx.PublicKey, tx.Data) {
		return fmt.Errorf("invalid transaction signature")
	}

	return nil
}
