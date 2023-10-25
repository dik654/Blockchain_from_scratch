package network

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// assert를 사용하기 위해
// go get github.com/stretchr/testify

// Makefile있는 디렉터리에서 make test로 테스트
func TestConnect(t *testing.T) {
	// LocalTransport 객체 주소 리턴
	tra := NewLocalTransport("A")
	trb := NewLocalTransport("B")

	// 락 걸고 peers mapping에 인수로 들어온 주소, LocalTransport 객체 매핑 쓰기
	tra.Connect(trb)
	trb.Connect(tra)
	// peers 매핑도 LocalTransport객체를 리턴하므로 trb와 동일한 객체를 갖는지 확인
	assert.Equal(t, tra.peers[trb.addr], trb)
	assert.Equal(t, trb.peers[tra.addr], tra)
}

func TestSendMessage(t *testing.T) {
	tra := NewLocalTransport("A")
	trb := NewLocalTransport("B")

	tra.Connect(trb)
	trb.Connect(tra)

	// B에게 consumeCh로 "hello world" 보내기
	msg := []byte("hello world")
	assert.Nil(t, tra.SendMessage(trb.addr, msg))

	// consumeCh 채널을 가져와서 데이터 consume하여
	// rpc 변수에 저장
	rpc := <-trb.Consume()
	// consume한 payload가 hello world인지
	assert.Equal(t, rpc.Payload, msg)
	// 송신자 주소가 A 주소가 맞는지 체크
	assert.Equal(t, rpc.From, tra.addr)
}
