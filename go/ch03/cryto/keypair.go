package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"

	"github.com/dik654/Blockchain_from_scratch/ch03/types"
)

type PrivateKey struct {
	key *ecdsa.PrivateKey
}

func GeneratePrivateKey() PrivateKey {
	// P256 곡선을 이용하여 개인키 생성
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	// 생성 실패 시 패닉
	if err != nil {
		panic(err)
	}

	// 성공시 개인 키 구조체 리턴
	return PrivateKey{
		key: key,
	}
}

func (k PrivateKey) PublicKey() PublicKey {
	return PublicKey{
		key: &k.key.PublicKey,
	}
}

type PublicKey struct {
	key *ecdsa.PublicKey
}

func (k PublicKey) Address() types.Address {
	return nil
}

type Signature struct {
}
