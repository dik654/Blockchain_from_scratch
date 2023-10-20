package main

import (
	"time"

	"github.com/dik654/Blockchain_from_scratch/go/ch01/01_transport_layer/network"
)

func main() {
	// 로컬, 원격 Transport 객체 생성하여 서로 연결
	trLocal := network.NewLocalTransport("LOCAL")
	trRemote := network.NewLocalTransport("REMOTE")

	trLocal.Connect(trRemote)
	trRemote.Connect(trLocal)

	// 고루틴을 생성하여
	go func() {
		for {
			// 원격 Transport에서 로컬 Transport로 1초마다 hello world를 계속 전송하도록 지시
			trRemote.SendMessage(trLocal.Addr(), []byte("hello world"))
			time.Sleep(1 * time.Second)
		}
	}()

	// 로컬 Transport 객체를 이용하여
	// ServerOpts 객체를 만들고
	opts := network.ServerOpts{
		Transports: []network.Transport{trLocal},
	}

	// ServerOpts를 이용하여 Server 객체 생성
	s := network.NewServer(opts)
	// 서버를 실행시켜 연결된 Transport들 채널에서 데이터 받아오기(1초마다 원격 Transport로부터 hello world를 받길 기대)
	// make run시 {From:REMOTE Payload:[104 101 108 108 111 32 119 111 114 108 100]}가 1초마다 콘솔에 뿌려져여함
	s.Start()
}
