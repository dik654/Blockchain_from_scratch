package types

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

type Hash [32]uint8

func (h Hash) IsZero() bool {
	// 해시된 값 32개의 uint8 값을 모두 읽어서 모두 0인지 체크
	for i := 0; i < 32; i++ {
		if h[i] != 0 {
			return false
		}
	}
	// 맞으면 true
	return true
}

func (h Hash) ToSlice() []byte {
	// 32bytes 버퍼 생성
	b := make([]byte, 32)
	for i := 0; i < 32; i++ {
		// [32]uint8 [32]bytes로 변환
		b[i] = h[i]
	}
	// 변환한 값 리턴
	return b
}

func (h Hash) String() string {
	// 32bytes값 string타입으로 인코딩
	return hex.EncodeToString(h.ToSlice())
}

func HashFromBytes(b []byte) Hash {
	// 들어온 byte수가 32개가 아니라면 에러
	if len(b) != 32 {
		msg := fmt.Sprintf("given bytes with length %d should be 32", len(b))
		panic(msg)
	}

	var value [32]uint8
	for i := 0; i < 32; i++ {
		value[i] = b[i]
	}
	// value 배열을 Hash타입([32]uint8)으로 변환
	return Hash(value)
}

// 생성할 bytes 길이 인수로 받기
func RandomBytes(size int) []byte {
	// 인수로 받은 길이만큼 슬라이스 생성
	token := make([]byte, size)
	// 슬라이스에 랜덤값 채워넣기
	rand.Read(token)
	// 슬라이스 리턴
	return token
}

func RandomHash() Hash {
	// 생성한 랜덤값을 [32]uint8 타입으로 변환해서 리턴
	return HashFromBytes(RandomBytes(32))
}
