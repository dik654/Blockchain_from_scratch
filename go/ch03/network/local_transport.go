package network

import (
	"fmt"
	"sync"
)

type LocalTransport struct {
	addr NetAddr
	// RPC 구조체 데이터가 이동하는 채널 생성
	consumeCh chan RPC
	// 동시성 문제 처리용
	lock sync.RWMutex
	// 해당 주소로 연결된 피어의 LocalTransport 객체 포인터 리턴
	peers map[NetAddr]*LocalTransport
}

func NewLocalTransport(addr NetAddr) Transport {
	// &로 생성한 객체 주소 리턴
	return &LocalTransport{
		addr: addr,
		// 1024개의 RPC 객체를 담을 수 있는 버퍼를 가진 채널 생성
		consumeCh: make(chan RPC, 1024),
		peers:     make(map[NetAddr]*LocalTransport),
	}
}

func (t *LocalTransport) Consume() <-chan RPC {
	// 다른 곳에서 사용할 수 있도록 채널 자체를 리턴
	return t.consumeCh
}

func (t *LocalTransport) Addr() NetAddr {
	return t.addr
}

func (t *LocalTransport) Connect(tr Transport) error {
	// 동시성 문제 때문에 함수 시작과 끝에 락을 걸고 푸는 과정을 둠
	// Lock을 실행한 곳에서만 뮤텍스의 데이터 읽기/쓰기 가능
	t.lock.Lock()
	defer t.lock.Unlock()

	t.peers[tr.Addr()] = tr.(*LocalTransport)

	return nil
}

// to 주소로 payload보내기
func (t *LocalTransport) SendMessage(to NetAddr, payload []byte) error {
	// Lock과 달리 RLock은 쓰는 것만 제한을 둠
	t.lock.RLock()
	defer t.lock.RUnlock()

	// 해당 주소가 연결된 피어인지 체크
	peer, ok := t.peers[to]
	// 연결되지 않았다면 오류 뿌리고 함수 종료
	if !ok {
		return fmt.Errorf("%s: could not send message to %s", t.addr, to)
	}

	// 연결 됐으면 consumeCh를 통해서 payload 보내기
	peer.consumeCh <- RPC{
		From:    t.addr,
		Payload: payload,
	}

	return nil
}
