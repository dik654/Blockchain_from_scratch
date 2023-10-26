package types

import (
	"encoding/hex"
	"fmt"
)

type Address [20]uint8

func (a Address) ToSlice() []byte {
	// uint8 배열을 bytes배열로 변환
	b := make([]byte, 20)
	for i := 0; i < 20; i++ {
		b[i] = a[i]
	}
	return b
}

// fmt.Println, Printf시 자동으로 실행
// bytes를 string으로 변환
func (a Address) String() string {
	return hex.EncodeToString(a.ToSlice())
}

func AddressFromBytes(b []byte) Address {
	// bytes 배열이 20보다 작으면 패닉
	if len(b) != 20 {
		msg := fmt.Sprintf("given bytes with length %d should be 20", len(b))
		panic(msg)
	}

	// bytes 배열을 uint8 배열로 변환
	var value [20]uint8
	for i := 0; i < 20; i++ {
		value[i] = b[i]
	}

	// 후 타입에 맞춰 선언하여 리턴
	return Address(value)
}
