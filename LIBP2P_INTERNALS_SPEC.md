# libp2p 내부 구조 완전 분석 (rust-libp2p)

> **목적**: libp2p의 전체 아키텍처를 실제 Rust 구현 코드를 바탕으로 100% 이해하기
> **대상**: rust-libp2p (https://github.com/libp2p/rust-libp2p)
> **버전**: v0.56.0+

## 목차

1. [전체 아키텍처 개요](#1-전체-아키텍처-개요)
2. [Core Layer - Transport & Muxer](#2-core-layer---transport--muxer)
3. [Transport 계층 구현](#3-transport-계층-구현)
4. [Stream Multiplexing](#4-stream-multiplexing)
5. [Swarm - 연결 관리](#5-swarm---연결-관리)
6. [NetworkBehaviour - 프로토콜 구현](#6-networkbehaviour---프로토콜-구현)
7. [Peer Discovery - Kad DHT](#7-peer-discovery---kad-dht)
8. [보안 계층 - Noise Protocol](#8-보안-계층---noise-protocol)
9. [실용 예제 코드](#9-실용-예제-코드)

---

## 1. 전체 아키텍처 개요

### 1.1 libp2p 철학

libp2p는 **모듈러 P2P 네트워킹 스택**으로, 각 레이어를 독립적으로 교체 가능하도록 설계되었습니다.

```
┌─────────────────────────────────────────┐
│   Application Protocols                 │  <- 사용자 정의 프로토콜
│   (Kad DHT, Gossipsub, Request/Response)│
├─────────────────────────────────────────┤
│   Swarm + NetworkBehaviour              │  <- 연결 관리 & 이벤트 핸들링
├─────────────────────────────────────────┤
│   Stream Multiplexing (Yamux, Mplex)    │  <- 멀티플렉싱
├─────────────────────────────────────────┤
│   Security Layer (Noise, TLS)           │  <- 암호화 & 인증
├─────────────────────────────────────────┤
│   Transport (TCP, QUIC, WebSocket)      │  <- 전송 계층
└─────────────────────────────────────────┘
```

### 1.2 디렉토리 구조

rust-libp2p 리포지토리 구조:

```
rust-libp2p/
├── core/                    # libp2p-core - 핵심 trait 정의
│   ├── src/
│   │   ├── transport/       # Transport trait
│   │   ├── muxing/          # StreamMuxer trait
│   │   ├── upgrade/         # 프로토콜 업그레이드
│   │   └── peer_id.rs       # PeerId (공개키 해시)
│
├── transports/              # 전송 프로토콜 구현
│   ├── tcp/                 # TCP transport
│   ├── quic/                # QUIC transport
│   ├── websocket/           # WebSocket transport
│   └── dns/                 # DNS 리졸빙
│
├── muxers/                  # 멀티플렉서 구현
│   ├── yamux/               # Yamux (권장)
│   └── mplex/               # Mplex
│
├── swarm/                   # libp2p-swarm - 연결 관리
│   ├── src/
│   │   ├── lib.rs           # Swarm 메인
│   │   ├── behaviour.rs     # NetworkBehaviour trait
│   │   ├── connection/      # 연결 핸들러
│   │   └── handler.rs       # ConnectionHandler
│
├── protocols/               # 애플리케이션 프로토콜
│   ├── kad/                 # Kademlia DHT
│   ├── gossipsub/           # Pub/Sub
│   ├── identify/            # Peer identification
│   ├── ping/                # Keepalive ping
│   ├── request-response/    # Generic RPC
│   └── mdns/                # Local discovery
│
└── misc/
    ├── noise/               # Noise 암호화
    └── plaintext/           # 평문 (테스트용)
```

---

## 2. Core Layer - Transport & Muxer

### 2.1 Transport Trait

**위치**: `core/src/transport/mod.rs`

Transport는 "어떻게 원격 노드에 도달할 것인가"를 정의합니다.

```rust
// core/src/transport/mod.rs

/// Transport trait은 연결을 생성하고 수신하는 방법을 정의
///
/// 핵심 개념:
/// - dial(): 원격 주소로 연결 시도
/// - listen_on(): 로컬 주소에서 연결 수신 대기
/// - 각 메서드는 Future를 반환하여 비동기 처리
pub trait Transport {
    /// 연결이 성공하면 생성되는 출력 타입
    /// 예: (PeerId, Stream) 또는 Muxer
    type Output;

    /// 다이얼 실패 시 에러 타입
    type Error: Error;

    /// dial()이 반환하는 Future
    type Dial: Future<Output = Result<Self::Output, Self::Error>>;

    /// listen_on()이 반환하는 리스너 스트림
    type ListenerUpgrade: Future<Output = Result<Self::Output, Self::Error>>;

    /// 원격 주소로 연결 시작
    ///
    /// 예: `/ip4/192.168.1.10/tcp/4001`
    fn dial(&mut self, addr: Multiaddr) -> Result<Self::Dial, TransportError<Self::Error>>;

    /// 로컬 주소에서 수신 대기
    ///
    /// 반환: ListenerStream (incoming connections)
    fn listen_on(
        &mut self,
        addr: Multiaddr,
    ) -> Result<Self::Listener, TransportError<Self::Error>>;

    /// Transport를 업그레이드 (암호화, 멀티플렉싱 추가)
    ///
    /// 이것이 libp2p의 핵심 설계 패턴!
    fn upgrade<U>(self, upgrade: U) -> Upgrade<Self, U>
    where
        Self: Sized,
        U: InboundUpgrade + OutboundUpgrade,
    {
        Upgrade::new(self, upgrade)
    }
}
```

**내부 로직 설명**:
- `dial()`: 주어진 Multiaddr을 파싱하여 연결 시도. TCP의 경우 OS socket API 호출
- `listen_on()`: OS에 바인딩하고 incoming 연결을 위한 스트림 반환
- `upgrade()`: **체인 패턴**으로 Transport를 감싸서 기능 추가 (암호화, 멀티플렉싱)

### 2.2 StreamMuxer Trait

**위치**: `core/src/muxing/mod.rs`

하나의 연결 위에서 여러 독립적인 스트림을 동시에 사용하기 위한 인터페이스입니다.

```rust
// core/src/muxing/mod.rs

/// StreamMuxer는 하나의 연결을 여러 substream으로 분할
///
/// 왜 필요한가?
/// - HTTP/2처럼 하나의 TCP 연결로 여러 요청 동시 처리
/// - 각 libp2p 프로토콜이 독립적인 스트림 사용 가능
///   (예: Kad DHT용 스트림, Gossipsub용 스트림, Ping용 스트림)
pub trait StreamMuxer {
    /// 개별 substream 타입
    type Substream: Stream;

    /// 에러 타입
    type Error: Error;

    /// 새로운 outbound 스트림 열기
    ///
    /// 용도: 이 노드가 먼저 프로토콜 시작할 때
    /// 예: DHT 쿼리를 보내거나, 메시지 publish
    fn poll_outbound(
        &self,
        cx: &mut Context<'_>,
    ) -> Poll<Result<Self::Substream, Self::Error>>;

    /// Inbound 스트림 수락
    ///
    /// 용도: 원격 피어가 시작한 프로토콜 처리
    /// 예: Ping 요청 받기, DHT 쿼리 응답
    fn poll_inbound(
        &self,
        cx: &mut Context<'_>,
    ) -> Poll<Result<Self::Substream, Self::Error>>;

    /// 연결 닫기
    fn poll_close(&self, cx: &mut Context<'_>) -> Poll<Result<(), Self::Error>>;
}
```

**내부 로직**:
- **Yamux/Mplex 내부**: 각 스트림에 고유 ID 부여, 프레임 헤더에 stream ID 포함
- **흐름 제어**: 각 스트림마다 독립적인 window 관리 (TCP의 receive window처럼)
- **우선순위**: Yamux는 스트림 우선순위 지원

---

## 3. Transport 계층 구현

### 3.1 TCP Transport

**위치**: `transports/tcp/src/lib.rs`

TCP는 가장 기본적인 전송 계층입니다.

```rust
// transports/tcp/src/lib.rs

use futures::prelude::*;
use std::net::SocketAddr;
use tokio::net::{TcpListener, TcpStream};

/// TCP Transport 구현체
///
/// 내부 상태:
/// - port_reuse: SO_REUSEPORT 활성화 여부
/// - nodelay: TCP_NODELAY (Nagle 알고리즘 비활성화)
pub struct TcpTransport {
    /// TCP_NODELAY 설정
    /// true면 작은 패킷도 즉시 전송 (지연 감소, P2P에 유리)
    nodelay: bool,

    /// SO_REUSEPORT 활성화
    /// 여러 리스너가 같은 포트 공유 가능
    port_reuse: bool,
}

impl TcpTransport {
    pub fn new(config: Config) -> Self {
        TcpTransport {
            nodelay: config.nodelay,
            port_reuse: config.port_reuse,
        }
    }
}

impl Transport for TcpTransport {
    type Output = TcpStream;
    type Error = io::Error;
    type Dial = Pin<Box<dyn Future<Output = Result<TcpStream, io::Error>> + Send>>;
    type Listener = TcpListenStream;

    /// TCP 연결 시작
    fn dial(&mut self, addr: Multiaddr) -> Result<Self::Dial, TransportError<Self::Error>> {
        // 1. Multiaddr를 SocketAddr로 변환
        //    예: /ip4/192.168.1.10/tcp/4001 -> 192.168.1.10:4001
        let socket_addr = multiaddr_to_socketaddr(&addr)?;

        // 2. TCP 연결 시도 (async)
        let nodelay = self.nodelay;
        let fut = async move {
            let stream = TcpStream::connect(socket_addr).await?;

            // 3. TCP_NODELAY 설정 (P2P는 지연 민감)
            if nodelay {
                stream.set_nodelay(true)?;
            }

            Ok(stream)
        };

        Ok(Box::pin(fut))
    }

    /// TCP 리스너 시작
    fn listen_on(
        &mut self,
        addr: Multiaddr,
    ) -> Result<Self::Listener, TransportError<Self::Error>> {
        let socket_addr = multiaddr_to_socketaddr(&addr)?;

        // 1. TcpListener 바인딩
        let listener = TcpListener::bind(socket_addr).await?;

        // 2. SO_REUSEPORT 설정 (여러 프로세스가 같은 포트 공유)
        if self.port_reuse {
            set_port_reuse(&listener)?;
        }

        // 3. 연결 수락 루프
        Ok(TcpListenStream { listener })
    }
}
```

**실제 내부 동작**:
1. `TcpStream::connect()`: Tokio의 async TCP, 내부적으로 epoll/kqueue 사용
2. `set_nodelay(true)`: Nagle 알고리즘 OFF → 작은 패킷도 즉시 전송 (P2P 메시지에 유리)
3. `SO_REUSEPORT`: Linux kernel 3.9+, 같은 포트에 여러 리스너 바인딩 가능

### 3.2 QUIC Transport

**위치**: `transports/quic/src/lib.rs`

QUIC은 UDP 기반으로 TLS 1.3 + 멀티플렉싱이 내장된 최신 프로토콜입니다.

```rust
// transports/quic/src/lib.rs

use quinn::{Endpoint, Connection};

/// QUIC Transport
///
/// 장점:
/// - 0-RTT 연결 재개 (이전 연결 정보로 즉시 데이터 전송)
/// - Head-of-line blocking 없음 (스트림 독립적)
/// - 내장 암호화 (TLS 1.3)
pub struct QuicTransport {
    /// Quinn endpoint (QUIC 구현체)
    endpoint: Endpoint,
}

impl Transport for QuicTransport {
    type Output = Connection;
    type Error = quinn::ConnectionError;

    fn dial(&mut self, addr: Multiaddr) -> Result<Self::Dial, TransportError<Self::Error>> {
        // 1. Multiaddr 파싱: /ip4/x.x.x.x/udp/port/quic-v1
        let socket_addr = parse_quic_multiaddr(&addr)?;

        // 2. QUIC 연결 시작
        //    내부: QUIC 핸드셰이크 (1-RTT 또는 0-RTT)
        //    - Initial 패킷 전송 (Client Hello 포함)
        //    - Handshake 패킷 교환 (TLS 1.3)
        //    - 1-RTT 패킷부터 데이터 전송 가능
        let connecting = self.endpoint.connect(socket_addr, "libp2p")?;

        Ok(async move {
            let connection = connecting.await?;
            Ok(connection)
        }.boxed())
    }
}
```

**QUIC vs TCP 비교**:
| 특징 | TCP | QUIC |
|------|-----|------|
| 연결 설정 | 3-way handshake (1 RTT) + TLS (1-2 RTT) | 1-RTT (또는 0-RTT) |
| 멀티플렉싱 | 별도 구현 필요 (Yamux) | 내장 (스트림 독립) |
| Head-of-line blocking | 있음 (packet loss 시 전체 대기) | 없음 (스트림별 독립) |
| 암호화 | 별도 TLS 필요 | 내장 (TLS 1.3) |

---

## 4. Stream Multiplexing

### 4.1 Yamux Multiplexer

**위치**: `muxers/yamux/src/lib.rs`

Yamux는 libp2p에서 권장하는 멀티플렉서입니다.

```rust
// muxers/yamux/src/lib.rs

/// Yamux 구성
pub struct Yamux {
    /// 설정: 최대 스트림 수, 윈도우 크기 등
    config: Config,
}

/// Yamux 프레임 헤더
///
/// 총 12바이트:
/// - version: 1바이트 (현재 0)
/// - type: 1바이트 (Data, WindowUpdate, Ping, GoAway)
/// - flags: 2바이트 (SYN, ACK, FIN, RST)
/// - stream_id: 4바이트 (스트림 식별자)
/// - length: 4바이트 (페이로드 크기)
struct Header {
    version: u8,
    ty: Type,
    flags: Flags,
    stream_id: u32,
    length: u32,
}

/// Yamux 연결 (하나의 TCP 연결 위에서 동작)
pub struct Connection<T> {
    /// 기본 TCP 스트림
    io: T,

    /// 열린 스트림들의 맵
    /// stream_id -> Stream
    streams: HashMap<u32, Stream>,

    /// 다음 outbound 스트림 ID (홀수)
    ///
    /// 왜 홀수?
    /// - 클라이언트: 홀수 (1, 3, 5, ...)
    /// - 서버: 짝수 (2, 4, 6, ...)
    /// -> ID 충돌 방지!
    next_outbound_id: u32,

    /// 전체 연결의 receive window
    ///
    /// 흐름 제어: 이 값이 0이면 상대방이 더 이상 데이터를 보낼 수 없음
    /// WindowUpdate 프레임으로 증가시킴
    window: u32,
}

impl<T: AsyncRead + AsyncWrite + Unpin> StreamMuxer for Yamux<T> {
    type Substream = Stream;

    /// 새 outbound 스트림 열기
    fn poll_outbound(&self, cx: &mut Context<'_>) -> Poll<Result<Self::Substream, Self::Error>> {
        // 1. 새 스트림 ID 할당
        let stream_id = self.next_outbound_id;
        self.next_outbound_id += 2;  // 홀수 유지

        // 2. SYN 프레임 전송
        //    Header { ty: Data, flags: SYN, stream_id, length: 0 }
        let header = Header::new(Type::Data, Flags::SYN, stream_id, 0);
        self.send_frame(header).await?;

        // 3. 스트림 생성 및 등록
        let stream = Stream::new(stream_id, self.config.receive_window);
        self.streams.insert(stream_id, stream.clone());

        Poll::Ready(Ok(stream))
    }

    /// Inbound 스트림 수락
    fn poll_inbound(&self, cx: &mut Context<'_>) -> Poll<Result<Self::Substream, Self::Error>> {
        // 1. 프레임 읽기 루프
        loop {
            let header = match self.read_frame(cx)? {
                Poll::Ready(h) => h,
                Poll::Pending => return Poll::Pending,
            };

            // 2. SYN 플래그 확인
            if header.flags.contains(Flags::SYN) {
                // 새 inbound 스트림!
                let stream = Stream::new(header.stream_id, self.config.receive_window);
                self.streams.insert(header.stream_id, stream.clone());

                // 3. SYN-ACK 응답
                let ack = Header::new(Type::Data, Flags::ACK, header.stream_id, 0);
                self.send_frame(ack).await?;

                return Poll::Ready(Ok(stream));
            }

            // 기타 프레임 처리 (Data, WindowUpdate 등)
            self.handle_frame(header)?;
        }
    }
}
```

**Yamux 스트림 흐름 제어**:

```rust
/// 개별 스트림
struct Stream {
    id: u32,

    /// 이 스트림의 receive window
    ///
    /// 의미: "내가 받을 수 있는 바이트 수"
    /// 0이 되면 상대방에게 WindowUpdate 요청
    recv_window: u32,

    /// 이 스트림의 send window
    ///
    /// 의미: "상대방이 받을 수 있는 바이트 수"
    /// 상대방의 WindowUpdate로 증가
    send_window: u32,

    /// 읽기 버퍼
    read_buffer: BytesMut,
}

impl Stream {
    /// 데이터 쓰기
    async fn write(&mut self, buf: &[u8]) -> io::Result<usize> {
        // 1. send_window 확인
        if self.send_window == 0 {
            // 상대방이 받을 수 없음 -> 대기
            return Err(io::ErrorKind::WouldBlock.into());
        }

        // 2. window 크기만큼만 전송
        let to_send = buf.len().min(self.send_window as usize);

        // 3. Data 프레임 전송
        let header = Header::new(Type::Data, Flags::empty(), self.id, to_send as u32);
        self.connection.send_frame_with_data(header, &buf[..to_send]).await?;

        // 4. send_window 감소
        self.send_window -= to_send as u32;

        Ok(to_send)
    }

    /// 데이터 읽기
    async fn read(&mut self, buf: &mut [u8]) -> io::Result<usize> {
        // 1. 버퍼에서 읽기
        let n = self.read_buffer.len().min(buf.len());
        buf[..n].copy_from_slice(&self.read_buffer.split_to(n));

        // 2. recv_window 업데이트
        //    읽은 만큼 다시 받을 수 있으므로 윈도우 증가
        self.recv_window += n as u32;

        // 3. WindowUpdate 전송 (상대방에게 알림)
        if self.recv_window > WINDOW_UPDATE_THRESHOLD {
            let header = Header::new(Type::WindowUpdate, Flags::empty(), self.id, n as u32);
            self.connection.send_frame(header).await?;
        }

        Ok(n)
    }
}
```

**왜 Yamux를 사용하는가?**
- HTTP/2의 멀티플렉싱과 유사한 설계
- 각 스트림 독립적 흐름 제어 (한 스트림이 막혀도 다른 스트림은 정상)
- 간단한 바이너리 프로토콜 (12바이트 헤더)

---

## 5. Swarm - 연결 관리

### 5.1 Swarm 구조

**위치**: `swarm/src/lib.rs`

Swarm은 libp2p의 핵심으로, 모든 연결과 프로토콜을 조율합니다.

```rust
// swarm/src/lib.rs

use futures::Stream;
use std::collections::HashMap;

/// Swarm: libp2p 네트워크의 중앙 관리자
///
/// 역할:
/// 1. Transport 관리 (TCP, QUIC 등)
/// 2. 연결 생성/종료
/// 3. NetworkBehaviour에게 이벤트 전달
/// 4. 프로토콜 협상 (Protocol Negotiation)
pub struct Swarm<TBehaviour>
where
    TBehaviour: NetworkBehaviour,
{
    /// 로컬 PeerId (이 노드의 공개키 해시)
    local_peer_id: PeerId,

    /// Transport (TCP, QUIC 등의 조합)
    /// 예: TCP + Noise + Yamux
    transport: Boxed<(PeerId, StreamMuxerBox)>,

    /// 활성화된 연결들
    ///
    /// 한 PeerId에 여러 연결 가능 (멀티어드레스)
    /// 예: 같은 피어에 TCP + QUIC 동시 연결
    connections: HashMap<PeerId, Vec<Connection>>,

    /// 사용자 정의 동작 (Kad DHT, Gossipsub 등)
    behaviour: TBehaviour,

    /// 대기 중인 이벤트 큐
    pending_events: VecDeque<SwarmEvent<TBehaviour::ToSwarm>>,
}

/// Swarm이 생성하는 이벤트
pub enum SwarmEvent<TBehaviourEvent> {
    /// 새 연결 확립
    ConnectionEstablished {
        peer_id: PeerId,
        endpoint: ConnectedPoint,
        /// 이 피어와의 연결 개수
        num_established: u32,
    },

    /// 연결 종료
    ConnectionClosed {
        peer_id: PeerId,
        cause: Option<ConnectionError>,
        num_established: u32,
    },

    /// Inbound 연결 (원격이 시작)
    IncomingConnection {
        local_addr: Multiaddr,
        send_back_addr: Multiaddr,
    },

    /// NetworkBehaviour에서 발생한 이벤트
    Behaviour(TBehaviourEvent),
}

impl<TBehaviour> Swarm<TBehaviour>
where
    TBehaviour: NetworkBehaviour,
{
    /// Swarm 생성
    pub fn new(
        transport: Boxed<(PeerId, StreamMuxerBox)>,
        behaviour: TBehaviour,
        local_peer_id: PeerId,
    ) -> Self {
        Swarm {
            local_peer_id,
            transport,
            connections: HashMap::new(),
            behaviour,
            pending_events: VecDeque::new(),
        }
    }

    /// 원격 피어로 연결
    ///
    /// 프로세스:
    /// 1. Transport.dial() 호출
    /// 2. 보안 핸드셰이크 (Noise)
    /// 3. 멀티플렉서 설정 (Yamux)
    /// 4. Identify 프로토콜 실행 (피어 정보 교환)
    pub fn dial(&mut self, addr: Multiaddr) -> Result<(), DialError> {
        // 1. Transport로 연결 시작
        let dial_fut = self.transport.dial(addr.clone())?;

        // 2. 비동기 태스크 생성
        let task = async move {
            // 2-1. 연결 완료 대기 (보안 + 멀티플렉싱 포함)
            let (peer_id, muxer) = dial_fut.await?;

            // 2-2. Connection 생성
            let connection = Connection::new(peer_id, muxer, ConnectionRole::Dialer);

            Ok((peer_id, connection))
        };

        // 3. 연결 추적
        self.pending_connections.push(task.boxed());

        Ok(())
    }

    /// 로컬 주소에서 수신 대기
    pub fn listen_on(&mut self, addr: Multiaddr) -> Result<ListenerId, TransportError> {
        // Transport 리스너 시작
        let listener = self.transport.listen_on(addr)?;

        let listener_id = ListenerId::new();
        self.listeners.insert(listener_id, listener);

        Ok(listener_id)
    }
}

/// Swarm을 Stream으로 사용 (이벤트 루프)
impl<TBehaviour> Stream for Swarm<TBehaviour>
where
    TBehaviour: NetworkBehaviour,
{
    type Item = SwarmEvent<TBehaviour::ToSwarm>;

    fn poll_next(mut self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        // 1. 대기 중인 이벤트가 있으면 즉시 반환
        if let Some(event) = self.pending_events.pop_front() {
            return Poll::Ready(Some(event));
        }

        // 2. Transport 이벤트 확인 (새 연결)
        for listener in &mut self.listeners.values_mut() {
            if let Poll::Ready(Some(event)) = listener.poll_next_unpin(cx) {
                match event {
                    ListenerEvent::Upgrade { upgrade, .. } => {
                        // 새 inbound 연결!
                        let (peer_id, muxer) = upgrade.await?;
                        let connection = Connection::new(peer_id, muxer, ConnectionRole::Listener);

                        self.connections.entry(peer_id).or_default().push(connection);

                        return Poll::Ready(Some(SwarmEvent::ConnectionEstablished {
                            peer_id,
                            endpoint: ConnectedPoint::Listener { .. },
                            num_established: self.connections[&peer_id].len() as u32,
                        }));
                    }
                    _ => {}
                }
            }
        }

        // 3. NetworkBehaviour poll
        //    사용자 정의 프로토콜 로직 실행
        if let Poll::Ready(event) = self.behaviour.poll(cx) {
            return Poll::Ready(Some(SwarmEvent::Behaviour(event)));
        }

        Poll::Pending
    }
}
```

**Swarm의 연결 라이프사이클**:

```
1. dial(addr) 호출
   ↓
2. Transport.dial() - TCP connect
   ↓
3. Noise 핸드셰이크 - 암호화 & 인증
   ↓
4. Yamux 설정 - 멀티플렉서 시작
   ↓
5. Identify 프로토콜 - 피어 정보 교환
   ↓
6. ConnectionEstablished 이벤트
   ↓
7. NetworkBehaviour.on_connection_established() 호출
   ↓
8. 애플리케이션 프로토콜 사용 가능 (Kad, Gossipsub 등)
```

---

## 6. NetworkBehaviour - 프로토콜 구현

### 6.1 NetworkBehaviour Trait

**위치**: `swarm/src/behaviour.rs`

NetworkBehaviour는 애플리케이션 프로토콜을 정의하는 핵심 인터페이스입니다.

```rust
// swarm/src/behaviour.rs

/// NetworkBehaviour: 피어와의 상호작용 로직 정의
///
/// 예시:
/// - Kad: DHT 쿼리/응답
/// - Gossipsub: 메시지 pub/sub
/// - Ping: keepalive
pub trait NetworkBehaviour {
    /// ConnectionHandler: 개별 연결의 프로토콜 처리
    type ConnectionHandler: ConnectionHandler;

    /// 이 Behaviour가 생성하는 이벤트 타입
    type ToSwarm: Debug;

    /// 새 연결 확립 시 호출
    ///
    /// 여기서:
    /// - 연결별 상태 초기화
    /// - ConnectionHandler 생성
    fn on_connection_established(
        &mut self,
        peer_id: PeerId,
        connection_id: ConnectionId,
        endpoint: &ConnectedPoint,
    ) -> Result<THandler, ConnectionDenied> {
        // 예: Kad의 경우 라우팅 테이블에 피어 추가
        // 예: Gossipsub의 경우 메시 토폴로지 업데이트
    }

    /// 연결 종료 시 호출
    fn on_connection_closed(
        &mut self,
        peer_id: PeerId,
        connection_id: ConnectionId,
        endpoint: &ConnectedPoint,
    ) {
        // 연결별 상태 정리
    }

    /// Swarm이 이 Behaviour를 폴링
    ///
    /// 반환: 발생한 이벤트 또는 Pending
    fn poll(
        &mut self,
        cx: &mut Context<'_>,
    ) -> Poll<ToSwarm<Self::ToSwarm, THandlerInEvent<Self>>> {
        // 주기적 작업 실행
        // 예: Kad - 라우팅 테이블 유지보수
        // 예: Gossipsub - heartbeat 전송
    }
}
```

### 6.2 NetworkBehaviour 조합 (Derive Macro)

여러 프로토콜을 동시에 사용하려면 조합이 필요합니다.

```rust
use libp2p::{kad, gossipsub, ping, identify};
use libp2p::swarm::NetworkBehaviour;

/// 여러 프로토콜을 조합한 Behaviour
///
/// Derive macro가 자동으로:
/// - 각 필드의 on_connection_established() 호출
/// - 각 필드의 poll() 순차 실행
/// - 이벤트를 MyBehaviourEvent enum으로 래핑
#[derive(NetworkBehaviour)]
#[behaviour(to_swarm = "MyBehaviourEvent")]
struct MyBehaviour {
    /// Kademlia DHT - 피어 발견 & 라우팅
    kad: kad::Behaviour<kad::store::MemoryStore>,

    /// Gossipsub - Pub/Sub 메시징
    gossipsub: gossipsub::Behaviour,

    /// Ping - 연결 keepalive
    ping: ping::Behaviour,

    /// Identify - 피어 정보 교환
    identify: identify::Behaviour,

    /// 필드 무시 (NetworkBehaviour 아님)
    #[behaviour(ignore)]
    custom_data: HashMap<PeerId, CustomData>,
}

/// 조합된 이벤트 enum (자동 생성)
#[derive(Debug)]
enum MyBehaviourEvent {
    Kad(kad::Event),
    Gossipsub(gossipsub::Event),
    Ping(ping::Event),
    Identify(identify::Event),
}
```

**내부 동작** (Derive macro 생성 코드):

```rust
impl NetworkBehaviour for MyBehaviour {
    // ...

    fn poll(&mut self, cx: &mut Context<'_>) -> Poll<...> {
        // 1. Kad poll
        if let Poll::Ready(event) = self.kad.poll(cx) {
            return Poll::Ready(ToSwarm::GenerateEvent(MyBehaviourEvent::Kad(event)));
        }

        // 2. Gossipsub poll
        if let Poll::Ready(event) = self.gossipsub.poll(cx) {
            return Poll::Ready(ToSwarm::GenerateEvent(MyBehaviourEvent::Gossipsub(event)));
        }

        // 3. Ping poll
        if let Poll::Ready(event) = self.ping.poll(cx) {
            return Poll::Ready(ToSwarm::GenerateEvent(MyBehaviourEvent::Ping(event)));
        }

        // 4. Identify poll
        if let Poll::Ready(event) = self.identify.poll(cx) {
            return Poll::Ready(ToSwarm::GenerateEvent(MyBehaviourEvent::Identify(event)));
        }

        Poll::Pending
    }
}
```

---

## 7. Peer Discovery - Kad DHT

### 7.1 Kademlia DHT 개요

**위치**: `protocols/kad/src/lib.rs`

Kademlia는 libp2p의 핵심 피어 발견 메커니즘입니다.

```rust
// protocols/kad/src/lib.rs

/// Kademlia DHT Behaviour
///
/// 기능:
/// 1. 피어 발견 (Peer Routing)
/// 2. 컨텐츠 발견 (Content Routing)
/// 3. 분산 저장소 (DHT)
pub struct Behaviour<TStore> {
    /// 로컬 PeerId
    local_peer_id: PeerId,

    /// K-bucket 라우팅 테이블
    ///
    /// Kademlia의 핵심 자료구조!
    /// - 거리별로 피어 분류 (XOR 거리)
    /// - 각 버킷에 k개 피어 (보통 k=20)
    kbuckets: KBucketsTable<PeerId, Addresses>,

    /// DHT 레코드 저장소
    store: TStore,

    /// 진행 중인 쿼리들
    queries: HashMap<QueryId, Query>,

    /// 설정 (복제 계수, 쿼리 병렬도 등)
    config: Config,
}

/// K-bucket 테이블
///
/// 구조:
/// - 256개의 버킷 (SHA-256 해시 기준)
/// - 버킷 인덱스 = distance의 leading zero bits
struct KBucketsTable<TKey, TVal> {
    /// 로컬 key (PeerId)
    local_key: TKey,

    /// 각 거리 범위별 버킷
    ///
    /// buckets[i]는 거리가 2^i ~ 2^(i+1) 범위인 피어들
    /// 예:
    /// - buckets[0]: 가장 가까운 피어들 (거리 1~2)
    /// - buckets[255]: 가장 먼 피어들
    buckets: Vec<KBucket<TVal>>,
}

impl<TStore> Behaviour<TStore> {
    /// 피어 찾기 쿼리 시작
    ///
    /// 프로세스:
    /// 1. 로컬 k-bucket에서 target에 가까운 k개 피어 선택
    /// 2. 그들에게 FIND_NODE 메시지 전송
    /// 3. 응답에서 더 가까운 피어 발견
    /// 4. 반복 (iterative lookup)
    pub fn find_peer(&mut self, target: PeerId) -> QueryId {
        // 1. 초기 피어 선택 (alpha개, 보통 3개)
        let closest = self.kbuckets.closest_keys(&target).take(self.config.alpha).collect();

        // 2. 쿼리 생성
        let query = Query {
            target,
            state: QueryState::WaitingAtClosest,
            peers: QueryPeers::new(closest),
        };

        let query_id = QueryId::new();
        self.queries.insert(query_id, query);

        // 3. FIND_NODE 요청 전송 (ConnectionHandler가 처리)
        for peer in &closest {
            self.send_request(*peer, KadRequestMsg::FindNode { key: target.into() });
        }

        query_id
    }

    /// FIND_NODE 응답 처리
    fn handle_find_node_response(&mut self, query_id: QueryId, closer_peers: Vec<PeerId>) {
        let query = self.queries.get_mut(&query_id).unwrap();

        // 1. 응답받은 피어들을 쿼리 상태에 추가
        for peer in closer_peers {
            if !query.peers.contains(&peer) {
                query.peers.insert(peer);

                // 2. 더 가까운 피어에게 계속 질의
                if query.peers.len() < self.config.k_value {
                    self.send_request(peer, KadRequestMsg::FindNode { key: query.target.into() });
                }
            }
        }

        // 3. 종료 조건 확인
        //    - k개 이상 피어 발견
        //    - 또는 더 가까운 피어 없음
        if query.is_finished() {
            let closest_peers = query.peers.closest_keys(&query.target).take(self.config.k_value).collect();

            // 이벤트 발생
            self.pending_events.push(KadEvent::FindPeerResult {
                peer: query.target,
                peers: closest_peers,
            });

            self.queries.remove(&query_id);
        }
    }
}
```

### 7.2 Kademlia 프로토콜 메시지

**위치**: `protocols/kad/src/protocol.rs`

```rust
// protocols/kad/src/protocol.rs

use prost::Message;  // Protocol Buffers

/// Kademlia 메시지 타입
#[derive(Clone, PartialEq, Message)]
pub struct KadMessage {
    /// 메시지 타입
    #[prost(enumeration = "MessageType", tag = "1")]
    pub type_: i32,

    /// 요청/응답 키
    #[prost(bytes, tag = "2")]
    pub key: Vec<u8>,

    /// 발견된 가까운 피어들
    #[prost(message, repeated, tag = "8")]
    pub closer_peers: Vec<Peer>,

    /// Provider 피어들 (content routing용)
    #[prost(message, repeated, tag = "9")]
    pub provider_peers: Vec<Peer>,
}

/// Peer 정보
#[derive(Clone, PartialEq, Message)]
pub struct Peer {
    /// PeerId (공개키 해시)
    #[prost(bytes, tag = "1")]
    pub id: Vec<u8>,

    /// 멀티어드레스들
    #[prost(bytes, repeated, tag = "2")]
    pub addrs: Vec<Vec<u8>>,

    /// 연결 타입 (connected, can_connect, cannot_connect)
    #[prost(enumeration = "ConnectionType", tag = "3")]
    pub connection: i32,
}

/// 메시지 타입 enum
#[derive(Clone, Copy, Debug, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub enum MessageType {
    /// 피어 찾기
    FindNode = 0,

    /// 값 저장
    PutValue = 1,

    /// 값 조회
    GetValue = 2,

    /// Provider 추가
    AddProvider = 3,

    /// Provider 조회
    GetProviders = 4,

    /// Ping
    Ping = 5,
}
```

**Kademlia 쿼리 예시**:

```
목표: PeerId = 0x1234...5678 찾기

1. 로컬 k-bucket 확인
   closest = [0x1200..., 0x1240..., 0x1260...]

2. FIND_NODE(0x1234...5678) 전송 → 3개 피어에게

3. 응답 수신:
   - 0x1200... → [0x1230..., 0x1231..., 0x1232...]
   - 0x1240... → [0x1234...5678]  ← 목표 발견!
   - 0x1260... → [0x1250..., 0x1255...]

4. 결과: 0x1234...5678의 주소 획득
```

---

## 8. 보안 계층 - Noise Protocol

### 8.1 Noise Protocol 개요

**위치**: `misc/noise/src/lib.rs`

Noise는 libp2p의 기본 암호화 프로토콜입니다.

```rust
// misc/noise/src/lib.rs

use snow::{Builder, HandshakeState};  // Noise protocol 구현

/// Noise 설정
///
/// libp2p는 Noise XX 패턴 사용:
/// - X: 정적 키 교환
/// - X: 양방향 인증
pub struct Config {
    /// 로컬 키쌍 (Ed25519 또는 X25519)
    keys: Keypair,

    /// Noise 패턴 (기본: XX)
    pattern: &'static str,
}

/// Noise 핸드셰이크 상태
pub struct NoiseHandshake {
    /// Snow의 핸드셰이크 상태 머신
    state: HandshakeState,

    /// 송신 버퍼
    send_buffer: Vec<u8>,

    /// 수신 버퍼
    recv_buffer: Vec<u8>,
}

impl Config {
    /// Noise 핸드셰이크 시작
    pub fn into_handshake(self, is_initiator: bool) -> NoiseHandshake {
        // 1. Noise 빌더 생성
        //    패턴: Noise_XX_25519_ChaChaPoly_BLAKE2s
        //    - XX: 양방향 인증
        //    - 25519: X25519 ECDH
        //    - ChaChaPoly: ChaCha20-Poly1305 AEAD
        //    - BLAKE2s: 해시 함수
        let builder = Builder::new("Noise_XX_25519_ChaChaPoly_BLAKE2s".parse().unwrap());

        // 2. 핸드셰이크 상태 생성
        let state = if is_initiator {
            builder
                .local_private_key(&self.keys.secret_key.as_bytes())
                .build_initiator()
                .unwrap()
        } else {
            builder
                .local_private_key(&self.keys.secret_key.as_bytes())
                .build_responder()
                .unwrap()
        };

        NoiseHandshake {
            state,
            send_buffer: vec![0u8; 65535],
            recv_buffer: vec![0u8; 65535],
        }
    }
}

impl NoiseHandshake {
    /// 핸드셰이크 실행 (XX 패턴)
    ///
    /// XX 패턴 메시지 흐름:
    /// 1. Initiator → Responder: e (ephemeral public key)
    /// 2. Responder → Initiator: e, ee, s (static public key)
    /// 3. Initiator → Responder: s, se
    pub async fn handshake<T: AsyncRead + AsyncWrite + Unpin>(
        mut self,
        io: &mut T,
    ) -> Result<(PeerId, NoiseOutput<T>), NoiseError> {
        if self.state.is_initiator() {
            // === Initiator 측 ===

            // 1. 메시지 1 전송: e
            let len = self.state.write_message(&[], &mut self.send_buffer)?;
            write_frame(io, &self.send_buffer[..len]).await?;

            // 2. 메시지 2 수신: e, ee, s
            let buf = read_frame(io).await?;
            let len = self.state.read_message(&buf, &mut self.recv_buffer)?;

            // 2-1. Responder의 static key 추출
            let remote_static = self.state.get_remote_static().unwrap();

            // 3. 메시지 3 전송: s, se
            let len = self.state.write_message(&[], &mut self.send_buffer)?;
            write_frame(io, &self.send_buffer[..len]).await?;

        } else {
            // === Responder 측 ===

            // 1. 메시지 1 수신: e
            let buf = read_frame(io).await?;
            self.state.read_message(&buf, &mut self.recv_buffer)?;

            // 2. 메시지 2 전송: e, ee, s
            let len = self.state.write_message(&[], &mut self.send_buffer)?;
            write_frame(io, &self.send_buffer[..len]).await?;

            // 3. 메시지 3 수신: s, se
            let buf = read_frame(io).await?;
            self.state.read_message(&buf, &mut self.recv_buffer)?;

            // 3-1. Initiator의 static key 추출
            let remote_static = self.state.get_remote_static().unwrap();
        }

        // 4. Transport mode로 전환
        //    이제 암호화된 데이터 전송 가능
        let transport = self.state.into_transport_mode()?;

        // 5. PeerId 검증
        //    상대방의 static key로 PeerId 계산
        let remote_peer_id = PeerId::from_public_key(&PublicKey::from_bytes(&remote_static)?);

        Ok((remote_peer_id, NoiseOutput::new(io, transport)))
    }
}

/// Noise 암호화 스트림
pub struct NoiseOutput<T> {
    /// 기본 TCP/QUIC 스트림
    io: T,

    /// Noise transport state (암호화/복호화)
    transport: TransportState,

    /// 암호화 버퍼
    encrypt_buffer: Vec<u8>,

    /// 복호화 버퍼
    decrypt_buffer: Vec<u8>,
}

impl<T: AsyncRead + AsyncWrite + Unpin> NoiseOutput<T> {
    /// 암호화된 데이터 쓰기
    async fn write(&mut self, plaintext: &[u8]) -> io::Result<usize> {
        // 1. ChaCha20-Poly1305로 암호화
        //    AEAD: plaintext + 16바이트 태그
        let len = self.transport.write_message(plaintext, &mut self.encrypt_buffer)?;

        // 2. 프레임 전송 (2바이트 길이 + 암호문)
        write_frame(&mut self.io, &self.encrypt_buffer[..len]).await?;

        Ok(plaintext.len())
    }

    /// 암호화된 데이터 읽기
    async fn read(&mut self, buf: &mut [u8]) -> io::Result<usize> {
        // 1. 프레임 수신
        let ciphertext = read_frame(&mut self.io).await?;

        // 2. 복호화 & 인증 검증
        let len = self.transport.read_message(&ciphertext, &mut self.decrypt_buffer)
            .map_err(|_| io::Error::new(io::ErrorKind::InvalidData, "decryption failed"))?;

        // 3. 평문 복사
        let to_copy = len.min(buf.len());
        buf[..to_copy].copy_from_slice(&self.decrypt_buffer[..to_copy]);

        Ok(to_copy)
    }
}
```

**Noise XX 핸드셰이크 상세**:

```
Initiator (Client)                 Responder (Server)
----------------                   ------------------

[generate ephemeral key e_i]

    --- e_i --->
                                   [generate ephemeral key e_r]
                                   [generate static key s_r]
                                   [compute: ee = DH(e_i, e_r)]
                                   [encrypt s_r with ee]

    <--- e_r, ee, s_r ---

[extract s_r]
[verify s_r signature]
[generate static key s_i]
[compute: se = DH(s_i, e_r)]
[encrypt s_i with ee+se]

    --- s_i, se --->
                                   [extract s_i]
                                   [verify s_i signature]
                                   [compute session keys]

[compute session keys]

=== 이제 암호화 통신 가능 ===
```

---

## 9. 실용 예제 코드

### 9.1 기본 libp2p 노드

```rust
// examples/basic_node.rs

use libp2p::{
    identity, ping, noise, tcp, yamux, Multiaddr, PeerId, Swarm,
    swarm::{NetworkBehaviour, SwarmEvent},
};
use std::error::Error;

/// 기본 Behaviour: Ping만 구현
#[derive(NetworkBehaviour)]
struct MyBehaviour {
    ping: ping::Behaviour,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn Error>> {
    // 1. 로컬 키쌍 생성 (Ed25519)
    //    이것으로 PeerId가 결정됨
    let local_key = identity::Keypair::generate_ed25519();
    let local_peer_id = PeerId::from(local_key.public());
    println!("Local PeerId: {}", local_peer_id);

    // 2. Transport 구성
    //    TCP + Noise 암호화 + Yamux 멀티플렉싱
    let transport = tcp::tokio::Transport::default()
        .upgrade(libp2p::core::upgrade::Version::V1)
        .authenticate(noise::Config::new(&local_key)?)
        .multiplex(yamux::Config::default())
        .boxed();

    // 3. Behaviour 생성
    let behaviour = MyBehaviour {
        ping: ping::Behaviour::new(ping::Config::new()),
    };

    // 4. Swarm 생성
    let mut swarm = Swarm::new(transport, behaviour, local_peer_id);

    // 5. 로컬 주소에서 수신 대기
    swarm.listen_on("/ip4/0.0.0.0/tcp/0".parse()?)?;

    // 6. 이벤트 루프
    loop {
        match swarm.select_next_some().await {
            SwarmEvent::NewListenAddr { address, .. } => {
                println!("Listening on {}", address);
            }
            SwarmEvent::Behaviour(event) => {
                println!("Ping event: {:?}", event);
            }
            SwarmEvent::ConnectionEstablished { peer_id, .. } => {
                println!("Connected to {}", peer_id);
            }
            SwarmEvent::ConnectionClosed { peer_id, cause, .. } => {
                println!("Disconnected from {}: {:?}", peer_id, cause);
            }
            _ => {}
        }
    }
}
```

### 9.2 Kademlia DHT 피어 발견

```rust
// examples/kad_peer_discovery.rs

use libp2p::{
    identity, kad, noise, tcp, yamux, Multiaddr, PeerId, Swarm,
    swarm::{NetworkBehaviour, SwarmEvent},
};
use std::error::Error;

#[derive(NetworkBehaviour)]
struct MyBehaviour {
    kad: kad::Behaviour<kad::store::MemoryStore>,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn Error>> {
    let local_key = identity::Keypair::generate_ed25519();
    let local_peer_id = PeerId::from(local_key.public());

    // Transport 구성
    let transport = tcp::tokio::Transport::default()
        .upgrade(libp2p::core::upgrade::Version::V1)
        .authenticate(noise::Config::new(&local_key)?)
        .multiplex(yamux::Config::default())
        .boxed();

    // Kad DHT 설정
    let mut kad_config = kad::Config::default();
    kad_config.set_query_timeout(std::time::Duration::from_secs(60));

    let store = kad::store::MemoryStore::new(local_peer_id);
    let mut kad = kad::Behaviour::with_config(local_peer_id, store, kad_config);

    // Bootstrap 노드 추가 (실제 libp2p bootstrap 노드)
    let bootaddr: Multiaddr = "/dnsaddr/bootstrap.libp2p.io".parse()?;
    kad.add_address(&"QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN".parse()?, bootaddr);

    let behaviour = MyBehaviour { kad };
    let mut swarm = Swarm::new(transport, behaviour, local_peer_id);

    swarm.listen_on("/ip4/0.0.0.0/tcp/0".parse()?)?;

    // Bootstrap 시작
    swarm.behaviour_mut().kad.bootstrap()?;

    loop {
        match swarm.select_next_some().await {
            SwarmEvent::Behaviour(MyBehaviourEvent::Kad(kad::Event::RoutingUpdated { peer, .. })) => {
                println!("Routing updated. Peer: {}", peer);
            }
            SwarmEvent::Behaviour(MyBehaviourEvent::Kad(kad::Event::OutboundQueryProgressed { result, .. })) => {
                match result {
                    kad::QueryResult::Bootstrap(Ok(kad::BootstrapOk { num_remaining, .. })) => {
                        println!("Bootstrap progress: {} remaining", num_remaining);
                        if num_remaining == 0 {
                            println!("Bootstrap complete!");

                            // 이제 피어 찾기 가능
                            let target = PeerId::random();
                            swarm.behaviour_mut().kad.get_closest_peers(target);
                        }
                    }
                    kad::QueryResult::GetClosestPeers(Ok(ok)) => {
                        println!("Found {} peers close to target", ok.peers.len());
                        for peer in ok.peers {
                            println!("  - {}", peer);
                        }
                    }
                    _ => {}
                }
            }
            _ => {}
        }
    }
}
```

### 9.3 Gossipsub Pub/Sub

```rust
// examples/gossipsub_chat.rs

use libp2p::{
    gossipsub, identity, noise, tcp, yamux, Multiaddr, PeerId, Swarm,
    swarm::{NetworkBehaviour, SwarmEvent},
};
use std::error::Error;
use std::time::Duration;

#[derive(NetworkBehaviour)]
struct MyBehaviour {
    gossipsub: gossipsub::Behaviour,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn Error>> {
    let local_key = identity::Keypair::generate_ed25519();
    let local_peer_id = PeerId::from(local_key.public());

    let transport = tcp::tokio::Transport::default()
        .upgrade(libp2p::core::upgrade::Version::V1)
        .authenticate(noise::Config::new(&local_key)?)
        .multiplex(yamux::Config::default())
        .boxed();

    // Gossipsub 설정
    let gossipsub_config = gossipsub::ConfigBuilder::default()
        .heartbeat_interval(Duration::from_secs(1))  // 피어 메시 유지 heartbeat
        .validation_mode(gossipsub::ValidationMode::Strict)  // 메시지 검증
        .build()
        .expect("Valid config");

    let mut gossipsub = gossipsub::Behaviour::new(
        gossipsub::MessageAuthenticity::Signed(local_key.clone()),
        gossipsub_config,
    )?;

    // 토픽 생성 및 구독
    let topic = gossipsub::IdentTopic::new("my-chat-room");
    gossipsub.subscribe(&topic)?;

    let behaviour = MyBehaviour { gossipsub };
    let mut swarm = Swarm::new(transport, behaviour, local_peer_id);

    swarm.listen_on("/ip4/0.0.0.0/tcp/0".parse()?)?;

    // 다른 피어에 연결 (예시)
    if let Some(remote) = std::env::args().nth(1) {
        let remote_addr: Multiaddr = remote.parse()?;
        swarm.dial(remote_addr)?;
    }

    // 주기적으로 메시지 발행
    let mut interval = tokio::time::interval(Duration::from_secs(10));

    loop {
        tokio::select! {
            _ = interval.tick() => {
                // 메시지 발행
                let msg = format!("Hello from {}", local_peer_id);
                swarm.behaviour_mut().gossipsub.publish(topic.clone(), msg.as_bytes())?;
                println!("Published: {}", msg);
            }

            event = swarm.select_next_some() => {
                match event {
                    SwarmEvent::Behaviour(MyBehaviourEvent::Gossipsub(gossipsub::Event::Message {
                        propagation_source,
                        message,
                        ..
                    })) => {
                        let msg = String::from_utf8_lossy(&message.data);
                        println!("Received from {}: {}", propagation_source, msg);
                    }
                    SwarmEvent::NewListenAddr { address, .. } => {
                        println!("Listening on {}", address);
                    }
                    _ => {}
                }
            }
        }
    }
}
```

---

## 10. 성능 최적화 및 베스트 프랙티스

### 10.1 Transport 선택

| Transport | 장점 | 단점 | 사용 사례 |
|-----------|------|------|-----------|
| **TCP** | 범용적, 방화벽 통과 용이 | 멀티플렉싱 별도 필요 | 일반 P2P 네트워크 |
| **QUIC** | 빠른 핸드셰이크, 내장 muxing | UDP 차단될 수 있음 | 고성능 요구 앱 |
| **WebSocket** | 브라우저 지원 | HTTP 오버헤드 | 웹 앱 연동 |

### 10.2 멀티플렉서 선택

- **Yamux**: 권장. 안정적이고 효율적
- **Mplex**: 레거시. Yamux로 마이그레이션 권장

### 10.3 연결 관리 최적화

```rust
use libp2p::swarm::ConnectionLimits;

let limits = ConnectionLimits::default()
    .with_max_pending_incoming(Some(10))      // 동시 pending 연결 제한
    .with_max_pending_outgoing(Some(20))
    .with_max_established_incoming(Some(100)) // 최대 inbound 연결
    .with_max_established_outgoing(Some(100))
    .with_max_established_per_peer(Some(2));  // 피어당 최대 연결 (멀티어드레스)

let swarm = SwarmBuilder::with_new_identity()
    .with_tokio()
    .with_tcp(...)
    .with_connection_limits(limits)
    .build();
```

### 10.4 메모리 최적화

```rust
// Kad DHT 메모리 제한
let mut kad_config = kad::Config::default();
kad_config.set_max_packet_size(16 * 1024);           // 패킷 크기 제한
kad_config.set_record_ttl(Some(Duration::from_secs(3600)));  // 레코드 TTL
kad_config.set_provider_record_ttl(Some(Duration::from_secs(3600)));
```

---

## 11. libp2p vs 다른 P2P 스택 비교

| 항목 | libp2p | DevP2P (Ethereum) | Custom P2P |
|------|--------|-------------------|------------|
| **모듈성** | ★★★★★ | ★★★☆☆ | ★☆☆☆☆ |
| **프로토콜 다양성** | 많음 (Kad, Gossipsub, etc) | 제한적 (RLPx 기반) | 직접 구현 |
| **NAT 트래버설** | ★★★★☆ (relay, hole punching) | ★★★☆☆ | 직접 구현 |
| **보안** | Noise, TLS | RLPx (ECIES) | 직접 구현 |
| **학습 곡선** | 중간 (trait 이해 필요) | 높음 (Ethereum 특화) | 낮음 (직접 제어) |
| **생태계** | 크림 (Polkadot, Filecoin 등) | Ethereum 전용 | 없음 |

---

## 12. 디버깅 및 모니터링

### 12.1 로깅 설정

```rust
use tracing_subscriber::{fmt, prelude::*, EnvFilter};

// 환경 변수로 로그 레벨 제어
// RUST_LOG=libp2p=debug,libp2p_gossipsub=trace cargo run
tracing_subscriber::registry()
    .with(fmt::layer())
    .with(EnvFilter::from_default_env())
    .init();
```

### 12.2 메트릭 수집

```rust
use libp2p::metrics::{Metrics, Recorder};
use prometheus::Registry;

let registry = Registry::new();
let metrics = Metrics::new(&registry);

// Swarm에 메트릭 recorder 추가
let swarm = SwarmBuilder::with_new_identity()
    .with_tokio()
    .with_tcp(...)
    .with_bandwidth_metrics(&metrics)
    .build();

// 주기적으로 메트릭 확인
println!("Connections: {}", registry.gather());
```

---

## 13. 다음 단계 학습 가이드

1. **소스 코드 읽기 순서**:
   ```
   1주차: core/src/transport, core/src/muxing
   2주차: transports/tcp, muxers/yamux
   3주차: swarm/src/behaviour.rs, swarm/src/lib.rs
   4주차: protocols/kad (Kademlia DHT)
   5주차: protocols/gossipsub (Pub/Sub)
   6주차: misc/noise (암호화)
   ```

2. **실습 프로젝트**:
   - **Week 1-2**: 기본 ping/pong 노드 구현
   - **Week 3-4**: Kad DHT로 피어 발견
   - **Week 5-6**: Gossipsub으로 채팅 앱
   - **Week 7-8**: Custom protocol 구현 (request/response)

3. **참고 자료**:
   - 공식 문서: https://docs.libp2p.io/
   - Rust API Docs: https://docs.rs/libp2p/
   - Spec: https://github.com/libp2p/specs
   - 예제 코드: https://github.com/libp2p/rust-libp2p/tree/master/examples

---

## 요약

libp2p는 **모듈러 설계**로 각 레이어를 독립적으로 선택/교체 가능한 P2P 네트워킹 스택입니다.

**핵심 개념**:
1. **Transport**: 연결 생성 방법 정의 (TCP, QUIC 등)
2. **Upgrade**: Transport에 기능 추가 (암호화, 멀티플렉싱) - 체인 패턴
3. **StreamMuxer**: 하나의 연결로 여러 스트림 (Yamux 권장)
4. **Swarm**: 모든 연결 관리 및 이벤트 조율
5. **NetworkBehaviour**: 애플리케이션 프로토콜 로직 (Kad, Gossipsub 등)

**실전 사용**:
- Polkadot, Filecoin, Ethereum 2.0 등 주요 블록체인 프로젝트가 사용
- Rust, Go, JavaScript 구현 모두 존재
- Production-ready, 활발한 커뮤니티

이 문서로 libp2p의 전체 내부 구조를 100% 이해하고, 직접 커스텀 P2P 애플리케이션을 구현할 수 있습니다.
