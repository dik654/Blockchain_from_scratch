package network

// 구조체에서의 가독성을 위해 선언
type NetAddr string

// Transport 위에서 동작
type RPC struct {
	From NetAddr
	// 인, 디코딩은 다른 레이어에서 처리할 예정
	Payload []byte
}

type Transport interface {
	Consume() <-chan RPC
	Connect(Transport) error
	SendMessage(NetAddr, []byte) error
	Addr() NetAddr
}
