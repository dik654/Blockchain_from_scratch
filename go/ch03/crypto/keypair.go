package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"math/big"

	"github.com/dik654/Blockchain_from_scratch/go/ch03/types"
)

type PrivateKey struct {
	key *ecdsa.PrivateKey
}

func (k PrivateKey) Sign(data []byte) (*Signature, error) {
	// 서명 rsv중 r, s값
	r, s, err := ecdsa.Sign(rand.Reader, k.key, data)
	if err != nil {
		return nil, err
	}

	return &Signature{
		r: r,
		s: s,
	}, nil
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

// 마샬링
func (k PublicKey) ToSlice() []byte {
	// 압축된 bytes 슬라이스로 변환
	// 변환된 값은 [Y값이 짝수일 경우 0x02 , 홀수일 경우 0x03] [X좌표 값]
	// k.key는 압축과정에서 곡선 정보를 얻는 용도
	return elliptic.MarshalCompressed(k.key, k.key.X, k.key.Y)
}

func (k PublicKey) Address() types.Address {
	// bytes 해싱
	h := sha256.Sum256(k.ToSlice())

	// 해시된 bytes의 끝 20자리를 uint8 배열로 변환
	return types.AddressFromBytes(h[len(h)-20:])
}

type Signature struct {
	s, r *big.Int
}

func (sig Signature) Verify(pubKey PublicKey, data []byte) bool {
	return ecdsa.Verify(pubKey.key, data, sig.r, sig.s)
}
