package network

import (
	"fmt"
	"time"
)

type ServerOpts struct {
	Transports []Transport
}

type Server struct {
	ServerOpts
	// 데이터 이동 채널
	rpcCh chan RPC
	// 서버 종료 채널
	quitCh chan struct{}
}

func NewServer(opts ServerOpts) *Server {
	return &Server{
		ServerOpts: opts,
		rpcCh:      make(chan RPC),
		quitCh:     make(chan struct{}, 1),
	}
}

func (s *Server) Start() {
	s.initTransports()
	// 5초 타이머
	// 종료되면 ticker.C 채널로 신호를 보냄
	ticker := time.NewTicker(5 * time.Second)
	// for문에 이름 붙이기
free:
	for {
		select {
		case rpc := <-s.rpcCh:
			// 필드 이름과 값 콘솔에 뿌리기
			fmt.Printf("%+v\n", rpc)
		// quitCh에 신호가 들어오면 Transport들에서 데이터를 읽어오는 이 free for문에서 탈출
		case <-s.quitCh:
			break free
			// default를 두어 select에서 대기하는 과정없이 계속 for루프를 돌게하는 것은 리소스를 많이 잡아먹으므로 Start함수에 타이머를 둔다
			// default:
		// 5초 동안 아무 데이터도 들어오지 않았다면
		case <-ticker.C:
			fmt.Printf("do stuff every x seconds")
		}
	}

	fmt.Println("Server shutdown")
}

func (s *Server) initTransports() {
	// 서버에 연결된 모든 Transport에 대하여
	for _, tr := range s.Transports {
		// 모두 고루틴을 띄워서
		go func(tr Transport) {
			// 각각의 Transport의 채널에서 데이터를 소비하여
			for rpc := range tr.Consume() {
				// 서버 rpcCh 채널로 데이터를 보낸다(각 Transport의 데이터를 한 곳으로 모으는 과정)
				s.rpcCh <- rpc
			}
		}(tr) // 고루틴으로 Transport 객체 전달
	}
}
