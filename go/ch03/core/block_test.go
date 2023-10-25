package core

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/dik654/Blockchain_from_scratch/go/ch02/types"
	"github.com/stretchr/testify/assert"
)

func TestHeader_Encode_Decode(t *testing.T) {
	// 테스트용 헤더 생성
	h := &Header{
		Version:   1,
		PrevBlock: types.RandomHash(),
		Timestamp: time.Now().UnixNano(),
		Height:    10,
		Nonce:     909834,
	}

	// bytes 버퍼 생성
	buf := &bytes.Buffer{}
	// 테스트용 헤더를 bytes로 인코딩하여 버퍼에 저장
	// 인코딩 중 에러가 나는지 체크
	assert.Nil(t, h.EncodeBinary(buf))

	// Header 객체를 저장할 시작 포인터를 저장
	hDecode := &Header{}
	// 버퍼에 저장된 bytes를 Header 객체 포인터에 차례대로 저장하여 디코딩
	// 디코딩 중 에러가 나는지 체크
	assert.Nil(t, hDecode.DecodeBinary(buf))
	// 전송했던 테스트용 헤더와 디코딩한 헤더가 동일한지 체크
	assert.Equal(t, h, hDecode)
}

func TestBlock_Encode_Decode(t *testing.T) {
	b := &Block{
		Header: Header{
			Version:   1,
			PrevBlock: types.RandomHash(),
			Timestamp: time.Now().UnixNano(),
			Height:    10,
			Nonce:     909834,
		},
		Transactions: nil,
	}

	// 헤더와 동일한 방식으로 테스팅
	buf := &bytes.Buffer{}
	assert.Nil(t, b.EncodeBinary(buf))

	bDecode := &Block{}
	assert.Nil(t, bDecode.DecodeBinary(buf))
	assert.Equal(t, b, bDecode)
}

func TestBlockHash(t *testing.T) {
	b := &Block{
		Header: Header{
			Version:   1,
			PrevBlock: types.RandomHash(),
			Timestamp: time.Now().UnixNano(),
			Height:    10,
			Nonce:     909834,
		},
		Transactions: []Transaction{},
	}

	h := b.Hash()
	fmt.Println(h)
	assert.False(t, h.IsZero())
}
