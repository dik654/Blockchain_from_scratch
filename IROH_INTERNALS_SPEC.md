# Iroh 내부 구조 완전 분석 (n0-computer/iroh)

> **목적**: Iroh의 전체 아키텍처를 실제 Rust 구현 코드를 바탕으로 100% 이해하기
> **대상**: n0-computer/iroh (https://github.com/n0-computer/iroh)
> **핵심**: QUIC 기반 P2P + BLAKE3 컨텐츠 주소 지정 + 자동 NAT 트래버설

## 목차

1. [전체 아키텍처 개요](#1-전체-아키텍처-개요)
2. [Endpoint - QUIC 연결 관리](#2-endpoint---quic-연결-관리)
3. [MagicEndpoint - NAT 트래버설](#3-magicendpoint---nat-트래버설)
4. [Relay 서버 구조](#4-relay-서버-구조)
5. [iroh-blobs - BLAKE3 컨텐츠 주소 지정](#5-iroh-blobs---blake3-컨텐츠-주소-지정)
6. [iroh-gossip - Pub/Sub 오버레이](#6-iroh-gossip---pubsub-오버레이)
7. [iroh-docs - CRDTs 문서 동기화](#7-iroh-docs---crdts-문서-동기화)
8. [보안 모델 - PublicKey 기반 인증](#8-보안-모델---publickey-기반-인증)
9. [실용 예제 코드](#9-실용-예제-코드)

---

## 1. 전체 아키텍처 개요

### 1.1 Iroh의 설계 철학

Iroh는 **"peer-2-peer that just works"**를 목표로 설계되었습니다.

핵심 특징:
- **QUIC 우선**: TCP 대신 QUIC으로 빠른 연결, 내장 암호화, 멀티플렉싱
- **자동 NAT 트래버설**: Relay + Hole punching 자동 처리
- **컨텐츠 주소 지정**: BLAKE3 해시로 데이터 검증
- **간단한 API**: libp2p보다 훨씬 단순한 인터페이스

```
┌──────────────────────────────────────────┐
│   Application Protocols                  │
│   (iroh-blobs, iroh-gossip, iroh-docs)   │
├──────────────────────────────────────────┤
│   Custom Protocol Layer                  │
│   (ALPN-based protocol routing)          │
├──────────────────────────────────────────┤
│   MagicEndpoint                          │
│   (NAT traversal, relay coordination)    │
├──────────────────────────────────────────┤
│   Quinn (QUIC Implementation)            │
│   (iroh-quinn fork with custom changes)  │
├──────────────────────────────────────────┤
│   UDP                                    │
└──────────────────────────────────────────┘
```

### 1.2 디렉토리 구조

```
iroh/
├── iroh/                           # 메인 크레이트
│   ├── src/
│   │   ├── endpoint.rs             # Endpoint + MagicEndpoint
│   │   ├── node.rs                 # 고수준 Node API
│   │   └── protocol.rs             # 프로토콜 라우터
│
├── iroh-net/                       # 네트워킹 레이어
│   ├── src/
│   │   ├── endpoint.rs             # MagicEndpoint 구현
│   │   ├── magicsock.rs            # Magic socket (relay + direct)
│   │   ├── relay/                  # Relay 프로토콜
│   │   ├── disco/                  # Discovery (STUN-like)
│   │   └── dns/                    # DNS discovery (n0 discovery)
│
├── iroh-blobs/                     # BLAKE3 blob 전송
│   ├── src/
│   │   ├── protocol.rs             # Blob 전송 프로토콜
│   │   ├── store/                  # Blob 저장소
│   │   ├── downloader.rs           # 다운로드 로직
│   │   └── provider.rs             # 제공자 로직
│
├── iroh-gossip/                    # Gossip 프로토콜
│   ├── src/
│   │   ├── proto.rs                # Gossip 메시지
│   │   └── net.rs                  # Gossip 네트워크
│
├── iroh-docs/                      # 문서 동기화 (CRDTs)
│   ├── src/
│   │   ├── engine.rs               # 동기화 엔진
│   │   ├── store/                  # 문서 저장소
│   │   └── sync.rs                 # 동기화 프로토콜
│
├── iroh-quinn/                     # Quinn fork
│   ├── src/
│   │   ├── endpoint.rs             # QUIC endpoint
│   │   └── connection.rs           # QUIC connection
│
└── iroh-relay/                     # Relay 서버
    ├── src/
    │   ├── server.rs               # Relay 서버 구현
    │   └── client.rs               # Relay 클라이언트
```

### 1.3 libp2p vs Iroh 비교

| 항목 | libp2p | Iroh |
|------|--------|------|
| **전송 계층** | TCP, QUIC, WebSocket 등 | QUIC 전용 |
| **멀티플렉싱** | Yamux, Mplex (별도) | QUIC 내장 |
| **암호화** | Noise, TLS (업그레이드) | QUIC TLS 1.3 (내장) |
| **NAT 트래버설** | 수동 설정 필요 | 자동 (relay + hole punching) |
| **피어 ID** | Multihash (여러 알고리즘) | Ed25519 공개키 (32바이트) |
| **API 복잡도** | 높음 (trait 많음) | 낮음 (간단한 struct) |
| **프로토콜 협상** | Multistream-select | ALPN (TLS 확장) |
| **주요 사용처** | Polkadot, Filecoin | 자체 프로젝트, 간단한 P2P 앱 |

---

## 2. Endpoint - QUIC 연결 관리

### 2.1 Endpoint 구조

**위치**: `iroh/src/endpoint.rs`

```rust
// iroh/src/endpoint.rs

use iroh_quinn as quinn;
use std::net::SocketAddr;

/// Iroh Endpoint: QUIC 연결의 진입점
///
/// 역할:
/// - QUIC 연결 생성 (dial)
/// - QUIC 연결 수락 (accept)
/// - NodeId 기반 주소 지정
pub struct Endpoint {
    /// Quinn의 QUIC endpoint
    ///
    /// Iroh는 Quinn을 fork하여 사용 (일부 커스터마이징)
    quinn_endpoint: quinn::Endpoint,

    /// 로컬 NodeId (Ed25519 공개키)
    ///
    /// 32바이트 공개키가 곧 노드 주소
    /// libp2p의 PeerId와 유사하지만 항상 Ed25519
    node_id: NodeId,

    /// Secret key (연결 인증용)
    secret_key: SecretKey,

    /// Home relay 서버 URL
    ///
    /// 기본값: https://relay.iroh.network
    /// NAT 뒤에 있을 때 relay를 통해 연결 시작
    relay_url: Option<RelayUrl>,

    /// 직접 연결 주소 (알려진 경우)
    ///
    /// STUN/disco를 통해 발견한 public 주소
    direct_addrs: Vec<SocketAddr>,
}

/// NodeId: 32바이트 Ed25519 공개키
///
/// 이것이 Iroh의 피어 주소!
/// 예: node_abc123...def (base32 인코딩)
#[derive(Clone, Copy, PartialEq, Eq, Hash)]
pub struct NodeId([u8; 32]);

impl Endpoint {
    /// Endpoint 생성 및 바인딩
    ///
    /// 프로세스:
    /// 1. Ed25519 키쌍 생성 (없으면)
    /// 2. QUIC endpoint 바인딩
    /// 3. Relay 서버 연결
    pub async fn bind() -> Result<Self, Error> {
        // 1. 키쌍 생성 또는 로드
        let secret_key = SecretKey::generate();
        let node_id = secret_key.public();

        // 2. TLS 설정 (QUIC는 TLS 1.3 필수)
        //    여기서 중요: 일반 TLS와 달리 도메인 이름 없음!
        //    대신 공개키를 TLS 인증서로 사용
        let tls_config = make_tls_config(&secret_key)?;

        // 3. QUIC endpoint 바인딩
        //    0.0.0.0:0 → OS가 자동으로 포트 할당
        let quinn_endpoint = quinn::Endpoint::server(tls_config, "0.0.0.0:0".parse()?)?;

        // 4. Relay URL 설정
        let relay_url = Some(RelayUrl::default());  // https://relay.iroh.network

        Ok(Endpoint {
            quinn_endpoint,
            node_id,
            secret_key,
            relay_url,
            direct_addrs: Vec::new(),
        })
    }

    /// NodeId로 연결
    ///
    /// Iroh의 핵심 API!
    /// 내부에서 자동으로:
    /// - Relay 통해 연결 시도
    /// - Hole punching으로 직접 연결 시도
    /// - 최적 경로 선택
    pub async fn connect(&self, node_id: NodeId) -> Result<Connection, Error> {
        // 실제 구현은 MagicEndpoint에서
        // 여기서는 간소화
        let addr = self.resolve_node_addr(node_id).await?;

        let connection = self.quinn_endpoint.connect(addr, "iroh")?.await?;

        Ok(Connection { inner: connection })
    }

    /// Incoming 연결 수락
    pub async fn accept(&self) -> Result<Connecting, Error> {
        let connecting = self.quinn_endpoint.accept().await
            .ok_or(Error::EndpointClosed)?;

        Ok(Connecting { inner: connecting })
    }

    /// 로컬 NodeId 반환
    pub fn node_id(&self) -> NodeId {
        self.node_id
    }

    /// 로컬 주소 반환 (UDP 바인딩 주소)
    pub fn local_addr(&self) -> SocketAddr {
        self.quinn_endpoint.local_addr().unwrap()
    }
}
```

**TLS 설정의 특이점**:

```rust
/// Iroh의 TLS 설정
///
/// 일반 TLS와 차이점:
/// - 도메인 이름 검증 없음
/// - 대신 공개키 검증 (self-signed)
/// - ALPN으로 프로토콜 협상
fn make_tls_config(secret_key: &SecretKey) -> Result<rustls::ServerConfig, Error> {
    // 1. Self-signed 인증서 생성
    //    Subject: CN=<node_id>
    //    Public Key: Ed25519 공개키
    let cert = generate_self_signed_cert(secret_key)?;

    // 2. 클라이언트 인증서 검증기 설정
    //    중요: 상대방의 공개키를 추출하여 NodeId 검증
    let verifier = CustomCertVerifier::new();

    // 3. ALPN 프로토콜 목록
    //    예: ["iroh-blobs/1", "iroh-gossip/0", "iroh-docs/0"]
    let alpn_protocols = vec![
        b"iroh-blobs/1".to_vec(),
        b"iroh-gossip/0".to_vec(),
    ];

    let mut config = rustls::ServerConfig::builder()
        .with_safe_defaults()
        .with_client_cert_verifier(Arc::new(verifier))
        .with_single_cert(vec![cert], secret_key.to_rustls_key())?;

    config.alpn_protocols = alpn_protocols;

    Ok(config)
}

/// 커스텀 인증서 검증기
///
/// 역할: 상대방의 Ed25519 공개키 추출 및 검증
struct CustomCertVerifier;

impl rustls::server::ClientCertVerifier for CustomCertVerifier {
    fn verify_client_cert(
        &self,
        end_entity: &rustls::Certificate,
        intermediates: &[rustls::Certificate],
        now: std::time::SystemTime,
    ) -> Result<rustls::server::ClientCertVerified, rustls::Error> {
        // 1. DER 인증서에서 공개키 추출
        let public_key = extract_ed25519_public_key(end_entity)?;

        // 2. NodeId 생성
        let node_id = NodeId::from(public_key);

        // 3. 검증 성공 (모든 self-signed 허용)
        //    실제 NodeId 검증은 애플리케이션 레벨에서
        Ok(rustls::server::ClientCertVerified::assertion())
    }
}
```

### 2.2 Connection 사용

```rust
/// QUIC Connection wrapper
pub struct Connection {
    inner: quinn::Connection,
}

impl Connection {
    /// 새 단방향 스트림 열기
    ///
    /// 용도: 요청 전송, 데이터 업로드
    pub async fn open_uni(&self) -> Result<SendStream, Error> {
        let send = self.inner.open_uni().await?;
        Ok(SendStream { inner: send })
    }

    /// 새 양방향 스트림 열기
    ///
    /// 용도: RPC, 요청-응답
    pub async fn open_bi(&self) -> Result<(SendStream, RecvStream), Error> {
        let (send, recv) = self.inner.open_bi().await?;
        Ok((SendStream { inner: send }, RecvStream { inner: recv }))
    }

    /// Incoming 스트림 수락
    pub async fn accept_bi(&self) -> Result<(SendStream, RecvStream), Error> {
        let (send, recv) = self.inner.accept_bi().await?;
        Ok((SendStream { inner: send }, RecvStream { inner: recv }))
    }

    /// 상대방 NodeId
    pub fn remote_node_id(&self) -> NodeId {
        // TLS 인증서에서 추출
        let peer_identity = self.inner.peer_identity().unwrap();
        extract_node_id_from_cert(peer_identity)
    }

    /// 연결 닫기
    pub fn close(&self, error_code: u64, reason: &[u8]) {
        self.inner.close(error_code.into(), reason);
    }
}
```

---

## 3. MagicEndpoint - NAT 트래버설

### 3.1 MagicEndpoint 개요

**위치**: `iroh-net/src/endpoint.rs`

MagicEndpoint는 Iroh의 핵심 혁신으로, **자동 NAT 트래버설**을 제공합니다.

```rust
// iroh-net/src/endpoint.rs

/// MagicEndpoint: Relay + Direct 연결 자동 관리
///
/// "Magic"의 의미:
/// - 사용자는 NodeId만 알면 됨
/// - NAT, 방화벽 신경 쓸 필요 없음
/// - 최적 경로 자동 선택
pub struct MagicEndpoint {
    /// 실제 QUIC endpoint
    endpoint: quinn::Endpoint,

    /// Magic socket (relay + direct 통합)
    ///
    /// 핵심: UDP socket 하나로 relay와 direct 동시 처리
    msock: MagicSocket,

    /// Home relay 클라이언트
    relay_client: Option<RelayClient>,

    /// Discovery 메커니즘 (STUN-like)
    disco: DiscoClient,

    /// 알려진 피어들의 주소 정보
    peer_map: PeerMap,
}

impl MagicEndpoint {
    /// MagicEndpoint 생성
    pub async fn bind() -> Result<Self, Error> {
        // 1. Secret key 생성
        let secret_key = SecretKey::generate();

        // 2. MagicSocket 생성
        //    이것이 Iroh의 핵심!
        let msock = MagicSocket::new(secret_key.clone()).await?;

        // 3. QUIC endpoint 생성 (MagicSocket 위에)
        let endpoint = quinn::Endpoint::new_with_abstract_socket(
            make_tls_config(&secret_key)?,
            None,
            Arc::new(msock.clone()),
        )?;

        // 4. Relay 클라이언트 시작
        let relay_url = RelayUrl::default();
        let relay_client = RelayClient::connect(relay_url, secret_key.clone()).await?;

        // 5. Discovery 클라이언트 시작
        let disco = DiscoClient::new(secret_key.public());

        Ok(MagicEndpoint {
            endpoint,
            msock,
            relay_client: Some(relay_client),
            disco,
            peer_map: PeerMap::new(),
        })
    }

    /// NodeId로 연결
    ///
    /// 내부 동작:
    /// 1. Relay를 통해 연결 시도 (즉시)
    /// 2. 동시에 직접 연결 시도 (hole punching)
    /// 3. 먼저 성공한 쪽 사용
    /// 4. Direct 성공 시 relay 중단
    pub async fn connect(&self, node_id: NodeId) -> Result<Connection, Error> {
        // 1. 피어 정보 조회 또는 생성
        let peer_state = self.peer_map.get_or_create(node_id);

        // 2. Relay를 통한 주소 구성
        //    예: relay:https://relay.iroh.network#<node_id>
        let relay_addr = self.relay_client.as_ref()
            .map(|c| c.relay_addr_for(node_id));

        // 3. 직접 주소 (알고 있다면)
        let direct_addrs = peer_state.direct_addrs();

        // 4. 병렬 연결 시도
        let connection = tokio::select! {
            // Relay 경로
            result = self.connect_via_relay(relay_addr) => result?,

            // Direct 경로 (hole punching)
            result = self.connect_direct(node_id, direct_addrs) => result?,
        };

        // 5. Direct 연결 성공 시 최적화
        if connection.is_direct() {
            // Relay 연결 정리
            peer_state.close_relay_connection();
        }

        Ok(connection)
    }

    /// Relay를 통한 연결
    async fn connect_via_relay(&self, relay_addr: SocketAddr) -> Result<Connection, Error> {
        // Relay 서버에 연결
        let connection = self.endpoint.connect(relay_addr, "iroh")?.await?;

        Ok(Connection {
            inner: connection,
            via_relay: true,
        })
    }

    /// 직접 연결 (hole punching)
    async fn connect_direct(&self, node_id: NodeId, addrs: Vec<SocketAddr>) -> Result<Connection, Error> {
        // 1. Discovery 요청 (STUN-like)
        //    Relay를 통해 상대방에게 disco 메시지 전송
        self.disco.send_disco_message(node_id, DiscoMessage::Ping).await?;

        // 2. Disco 응답 대기 (상대방의 주소 획득)
        let peer_addr = tokio::time::timeout(
            Duration::from_secs(5),
            self.disco.wait_for_pong(node_id)
        ).await??;

        // 3. UDP hole punching
        //    양쪽에서 동시에 UDP 패킷 전송
        //    -> NAT에 매핑 생성
        self.msock.send_disco_packet(peer_addr, DiscoMessage::Ping).await?;

        // 4. QUIC 연결 시도
        let connection = self.endpoint.connect(peer_addr, "iroh")?.await?;

        Ok(Connection {
            inner: connection,
            via_relay: false,
        })
    }
}
```

### 3.2 MagicSocket 구조

**위치**: `iroh-net/src/magicsock.rs`

MagicSocket은 **하나의 UDP socket으로 relay와 direct 패킷을 동시에 처리**합니다.

```rust
// iroh-net/src/magicsock.rs

use tokio::net::UdpSocket;

/// MagicSocket: Relay + Direct 통합 소켓
///
/// 핵심 아이디어:
/// - Relay 패킷: 특별한 헤더로 식별
/// - Direct 패킷: 일반 QUIC 패킷
/// - 하나의 UDP socket에서 둘 다 처리
pub struct MagicSocket {
    /// 실제 UDP socket
    ///
    /// 바인딩: 0.0.0.0:<random_port>
    udp_socket: Arc<UdpSocket>,

    /// Relay 연결 상태
    relay_conns: HashMap<RelayUrl, RelayConn>,

    /// Direct 주소 매핑
    ///
    /// NodeId -> SocketAddr
    /// Disco를 통해 획득한 주소들
    peer_addrs: HashMap<NodeId, SocketAddr>,

    /// 수신 패킷 버퍼
    recv_buffer: BytesMut,
}

impl MagicSocket {
    /// 패킷 수신 (AsyncUdpSocket trait)
    ///
    /// Quinn이 호출하는 메서드
    async fn recv_from(&self, buf: &mut [u8]) -> io::Result<(usize, SocketAddr)> {
        loop {
            // 1. UDP 패킷 수신
            let (len, src_addr) = self.udp_socket.recv_from(buf).await?;

            // 2. 패킷 타입 판별
            if is_relay_packet(&buf[..len]) {
                // Relay 패킷 처리
                let (payload, node_id) = self.unwrap_relay_packet(&buf[..len])?;

                // Relay 헤더 제거 후 페이로드 반환
                buf[..payload.len()].copy_from_slice(payload);

                // 논리적 src_addr: relay 주소 대신 NodeId 기반 주소 반환
                let logical_addr = self.node_id_to_addr(node_id);
                return Ok((payload.len(), logical_addr));

            } else {
                // Direct 패킷: 그대로 반환
                return Ok((len, src_addr));
            }
        }
    }

    /// 패킷 전송 (AsyncUdpSocket trait)
    async fn send_to(&self, buf: &[u8], target: SocketAddr) -> io::Result<usize> {
        // 1. 타겟이 relay 주소인지 확인
        if let Some(node_id) = self.addr_to_node_id(target) {
            // 1-1. Relay를 통해 전송
            if let Some(relay_conn) = self.get_relay_for_node(node_id) {
                return relay_conn.send_to_node(node_id, buf).await;
            }

            // 1-2. Direct 주소가 있으면 직접 전송
            if let Some(direct_addr) = self.peer_addrs.get(&node_id) {
                return self.udp_socket.send_to(buf, direct_addr).await;
            }

            return Err(io::Error::new(io::ErrorKind::NotFound, "no route to node"));
        }

        // 2. 일반 주소: 직접 전송
        self.udp_socket.send_to(buf, target).await
    }
}

/// Relay 패킷 판별
///
/// Relay 패킷 포맷:
/// - Magic: 4바이트 (0x52, 0x4C, 0x59, 0x00) "RLY\0"
/// - NodeId: 32바이트 (발신자)
/// - Payload: QUIC 패킷
fn is_relay_packet(buf: &[u8]) -> bool {
    buf.len() > 4 && &buf[..4] == b"RLY\0"
}
```

**MagicSocket의 작동 방식**:

```
시나리오: Node A가 Node B에 연결

1. 초기 상태:
   - A: NAT 뒤 (private IP)
   - B: NAT 뒤 (private IP)
   - Relay: public IP

2. A가 B에 connect() 호출:

   Step 1: Relay 경로 (즉시 시작)
   A → [Relay] → B

   패킷 구조:
   [RLY\0][NodeId(A)][QUIC Initial Packet]

   Relay는 NodeId(B)를 보고 B에게 전달:
   [RLY\0][NodeId(A)][QUIC Initial Packet]

   Step 2: Direct 경로 (병렬 시도)
   A → Relay: "Disco Ping to B"
   Relay → B: [RLY\0][NodeId(A)][Disco Ping]

   B → Relay: "Disco Pong with my public IP"
   Relay → A: [RLY\0][NodeId(B)][Disco Pong: B_public_IP]

   Step 3: Hole Punching
   A → B_public_IP: [QUIC Packet]  (NAT에 매핑 생성)
   B → A_public_IP: [QUIC Packet]  (NAT에 매핑 생성)

   Step 4: Direct 연결 성공!
   A ↔ B (직접 UDP)

   Relay 연결 종료

3. 결과:
   - Fast path: Direct UDP (낮은 지연)
   - Fallback: Relay (항상 작동)
```

---

## 4. Relay 서버 구조

### 4.1 Relay 프로토콜

**위치**: `iroh-relay/src/server.rs`

```rust
// iroh-relay/src/server.rs

use tokio::net::TcpListener;
use futures::StreamExt;

/// Relay 서버
///
/// 역할:
/// - NAT 뒤 피어들의 중계 서버
/// - NodeId 기반 라우팅
/// - Disco 메시지 중계
pub struct RelayServer {
    /// 리스닝 주소 (예: 0.0.0.0:443)
    bind_addr: SocketAddr,

    /// 현재 연결된 클라이언트들
    ///
    /// NodeId -> RelayClientConn
    clients: HashMap<NodeId, RelayClientConn>,

    /// 통계
    stats: RelayStats,
}

/// 클라이언트 연결
struct RelayClientConn {
    /// 클라이언트 NodeId
    node_id: NodeId,

    /// WebSocket 연결
    ///
    /// Relay는 WebSocket over HTTPS 사용
    /// 이유: 방화벽 통과 용이
    ws: WebSocketStream<TcpStream>,

    /// 마지막 활동 시간
    last_active: Instant,
}

impl RelayServer {
    /// Relay 서버 시작
    pub async fn run(bind_addr: SocketAddr) -> Result<(), Error> {
        let listener = TcpListener::bind(bind_addr).await?;
        println!("Relay server listening on {}", bind_addr);

        let mut clients = HashMap::new();

        loop {
            let (stream, peer_addr) = listener.accept().await?;

            // WebSocket 업그레이드
            let ws = tokio_tungstenite::accept_async(stream).await?;

            // 클라이언트 핸들러 spawn
            tokio::spawn(Self::handle_client(ws, peer_addr, clients.clone()));
        }
    }

    /// 클라이언트 핸들러
    async fn handle_client(
        mut ws: WebSocketStream<TcpStream>,
        peer_addr: SocketAddr,
        clients: Arc<RwLock<HashMap<NodeId, RelayClientConn>>>,
    ) -> Result<(), Error> {
        // 1. 첫 메시지: ClientInfo (NodeId 포함)
        let msg = ws.next().await.ok_or(Error::ConnectionClosed)??;
        let client_info: ClientInfo = bincode::deserialize(&msg.into_data())?;

        let node_id = client_info.node_id;
        println!("Client connected: {}", node_id);

        // 2. 클라이언트 등록
        let conn = RelayClientConn {
            node_id,
            ws: ws.clone(),
            last_active: Instant::now(),
        };
        clients.write().await.insert(node_id, conn);

        // 3. 메시지 루프
        while let Some(result) = ws.next().await {
            let msg = result?;

            match msg {
                Message::Binary(data) => {
                    // 바이너리 메시지: Relay 패킷
                    // 포맷: [target_node_id: 32bytes][payload]
                    if data.len() < 32 {
                        continue;
                    }

                    let target_node_id = NodeId::from_bytes(&data[..32]);
                    let payload = &data[32..];

                    // 타겟 클라이언트에게 전달
                    if let Some(target_conn) = clients.read().await.get(&target_node_id) {
                        // Relay 헤더 추가: [RLY\0][sender_node_id][payload]
                        let mut relayed = Vec::with_capacity(4 + 32 + payload.len());
                        relayed.extend_from_slice(b"RLY\0");
                        relayed.extend_from_slice(node_id.as_bytes());
                        relayed.extend_from_slice(payload);

                        target_conn.ws.send(Message::Binary(relayed)).await?;
                    }
                }

                Message::Ping(_) => {
                    // Keepalive
                    ws.send(Message::Pong(vec![])).await?;
                }

                Message::Close(_) => {
                    break;
                }

                _ => {}
            }
        }

        // 4. 클라이언트 제거
        clients.write().await.remove(&node_id);
        println!("Client disconnected: {}", node_id);

        Ok(())
    }
}
```

### 4.2 Relay 클라이언트

**위치**: `iroh-relay/src/client.rs`

```rust
// iroh-relay/src/client.rs

/// Relay 클라이언트
///
/// Endpoint와 함께 실행되어 relay 연결 유지
pub struct RelayClient {
    /// Relay 서버 URL
    relay_url: RelayUrl,

    /// 로컬 NodeId
    node_id: NodeId,

    /// WebSocket 연결
    ws: WebSocketStream<MaybeTlsStream<TcpStream>>,

    /// 송신 채널
    ///
    /// Endpoint가 relay로 패킷 전송할 때 사용
    send_tx: mpsc::Sender<RelayMessage>,

    /// 수신 채널
    ///
    /// Relay에서 받은 패킷을 MagicSocket에 전달
    recv_rx: mpsc::Receiver<(NodeId, Bytes)>,
}

impl RelayClient {
    /// Relay 서버 연결
    pub async fn connect(relay_url: RelayUrl, node_id: NodeId) -> Result<Self, Error> {
        // 1. WebSocket 연결
        let (ws_stream, _) = tokio_tungstenite::connect_async(relay_url.ws_url()).await?;

        // 2. ClientInfo 전송
        let client_info = ClientInfo { node_id };
        let msg = Message::Binary(bincode::serialize(&client_info)?);
        ws_stream.send(msg).await?;

        // 3. 채널 생성
        let (send_tx, send_rx) = mpsc::channel(100);
        let (recv_tx, recv_rx) = mpsc::channel(100);

        // 4. 수신 루프 spawn
        tokio::spawn(Self::recv_loop(ws_stream.clone(), recv_tx));

        // 5. 송신 루프 spawn
        tokio::spawn(Self::send_loop(ws_stream.clone(), send_rx));

        Ok(RelayClient {
            relay_url,
            node_id,
            ws: ws_stream,
            send_tx,
            recv_rx,
        })
    }

    /// 노드에게 패킷 전송
    pub async fn send_to_node(&self, target: NodeId, payload: &[u8]) -> Result<(), Error> {
        // Relay 메시지 구성: [target_node_id][payload]
        let mut msg = Vec::with_capacity(32 + payload.len());
        msg.extend_from_slice(target.as_bytes());
        msg.extend_from_slice(payload);

        self.send_tx.send(RelayMessage::Data(msg)).await?;
        Ok(())
    }

    /// 수신 루프
    async fn recv_loop(
        mut ws: WebSocketStream<MaybeTlsStream<TcpStream>>,
        recv_tx: mpsc::Sender<(NodeId, Bytes)>,
    ) {
        while let Some(Ok(msg)) = ws.next().await {
            if let Message::Binary(data) = msg {
                // Relay 패킷 파싱: [RLY\0][sender_node_id][payload]
                if data.len() < 36 || &data[..4] != b"RLY\0" {
                    continue;
                }

                let sender = NodeId::from_bytes(&data[4..36]);
                let payload = Bytes::copy_from_slice(&data[36..]);

                let _ = recv_tx.send((sender, payload)).await;
            }
        }
    }

    /// 송신 루프
    async fn send_loop(
        mut ws: WebSocketStream<MaybeTlsStream<TcpStream>>,
        mut send_rx: mpsc::Receiver<RelayMessage>,
    ) {
        while let Some(msg) = send_rx.recv().await {
            match msg {
                RelayMessage::Data(data) => {
                    let _ = ws.send(Message::Binary(data)).await;
                }
            }
        }
    }
}
```

**Relay의 성능 최적화**:
- WebSocket over HTTPS: 방화벽 통과 용이
- Binary 프로토콜: 오버헤드 최소화
- Zero-copy 전달: 패킷을 파싱하지 않고 바로 전달
- 클라이언트 hashmap: O(1) 라우팅

---

## 5. iroh-blobs - BLAKE3 컨텐츠 주소 지정

### 5.1 BLAKE3 해시와 Verified Streaming

**위치**: `iroh-blobs/src/protocol.rs`

iroh-blobs는 **BLAKE3 verified streaming**으로 데이터 무결성을 보장합니다.

```rust
// iroh-blobs/src/protocol.rs

use blake3::Hash as Blake3Hash;

/// Blob 해시 (32바이트 BLAKE3)
///
/// 이것이 컨텐츠 주소!
/// 예: blake3:abc123...def
#[derive(Clone, Copy, PartialEq, Eq, Hash)]
pub struct Hash(Blake3Hash);

/// Blob 요청
///
/// 프로토콜: iroh-blobs/1 (ALPN)
#[derive(Debug, Clone)]
pub struct GetRequest {
    /// 요청할 blob 해시
    pub hash: Hash,

    /// 바이트 범위 (optional)
    ///
    /// 예: Some(1024..2048) - 1KB부터 2KB까지만
    pub range: Option<Range<u64>>,
}

/// Blob 응답
#[derive(Debug)]
pub struct GetResponse {
    /// 전체 크기
    pub size: u64,

    /// 데이터 스트림 (verified streaming)
    pub data: VerifiedStream,
}

/// BLAKE3 Verified Stream
///
/// 핵심: 스트리밍하면서 동시에 해시 검증
///
/// BLAKE3의 특징:
/// - Incremental hashing (청크 단위)
/// - Tree structure (병렬 검증 가능)
pub struct VerifiedStream {
    /// 기대하는 최종 해시
    expected_hash: Hash,

    /// BLAKE3 hasher (incremental)
    hasher: blake3::Hasher,

    /// 데이터 스트림
    inner: Box<dyn AsyncRead + Unpin + Send>,

    /// 현재까지 읽은 바이트 수
    bytes_read: u64,
}

impl VerifiedStream {
    /// 새 verified stream 생성
    pub fn new(expected_hash: Hash, data: impl AsyncRead + Unpin + Send + 'static) -> Self {
        VerifiedStream {
            expected_hash,
            hasher: blake3::Hasher::new(),
            inner: Box::new(data),
            bytes_read: 0,
        }
    }

    /// 데이터 읽기 (AsyncRead impl)
    async fn read(&mut self, buf: &mut [u8]) -> io::Result<usize> {
        // 1. 데이터 읽기
        let n = self.inner.read(buf).await?;

        if n == 0 {
            // EOF: 최종 해시 검증
            let computed_hash = self.hasher.finalize();

            if computed_hash != self.expected_hash.0 {
                return Err(io::Error::new(
                    io::ErrorKind::InvalidData,
                    format!("Hash mismatch: expected {}, got {}", self.expected_hash, computed_hash),
                ));
            }

            return Ok(0);
        }

        // 2. Incremental 해시 업데이트
        //    BLAKE3는 청크 단위로 해시 계산 가능
        self.hasher.update(&buf[..n]);
        self.bytes_read += n as u64;

        Ok(n)
    }
}
```

**BLAKE3 Verified Streaming의 장점**:

```
기존 방식 (전체 다운로드 후 검증):
1. 10GB 파일 전체 다운로드
2. SHA256 계산 (느림)
3. 검증 실패 시 처음부터 재다운로드

BLAKE3 Verified Streaming:
1. 데이터 읽으면서 동시에 해시 계산 (빠름)
2. EOF에서 즉시 검증
3. 실패 시 즉시 중단 (대역폭 절약)
4. 청크 단위 병렬 검증 가능 (멀티코어 활용)
```

### 5.2 Blob 다운로드 프로토콜

```rust
// iroh-blobs/src/downloader.rs

/// Blob 다운로더
pub struct Downloader {
    /// Endpoint
    endpoint: Endpoint,

    /// 로컬 저장소
    store: Store,

    /// 진행 중인 다운로드
    active_downloads: HashMap<Hash, DownloadTask>,
}

impl Downloader {
    /// Blob 다운로드
    ///
    /// 프로세스:
    /// 1. Provider 노드 발견
    /// 2. 연결 및 GET 요청
    /// 3. Verified streaming으로 수신
    /// 4. 로컬 저장소에 저장
    pub async fn download(&self, hash: Hash, providers: Vec<NodeId>) -> Result<(), Error> {
        // 1. Provider 선택 (가장 가까운 것)
        let provider = providers.first().ok_or(Error::NoProviders)?;

        // 2. Provider에 연결
        let connection = self.endpoint.connect(*provider).await?;

        // 3. iroh-blobs 프로토콜로 양방향 스트림 열기
        let (mut send, mut recv) = connection.open_bi().await?;

        // 4. GET 요청 전송
        let request = GetRequest {
            hash,
            range: None,  // 전체 다운로드
        };
        send.write_all(&bincode::serialize(&request)?).await?;
        send.finish().await?;

        // 5. 응답 수신
        let response: GetResponse = bincode::deserialize_from(&mut recv)?;

        // 6. Verified streaming으로 데이터 수신
        let mut verified_stream = VerifiedStream::new(hash, recv);

        // 7. 로컬 저장소에 저장
        let mut writer = self.store.create_blob(hash).await?;

        let mut buf = vec![0u8; 64 * 1024];  // 64KB 버퍼
        loop {
            let n = verified_stream.read(&mut buf).await?;
            if n == 0 {
                break;  // EOF, 해시 검증 성공!
            }

            writer.write_all(&buf[..n]).await?;
        }

        writer.finish().await?;
        println!("Downloaded blob: {}", hash);

        Ok(())
    }
}
```

### 5.3 Blob 제공 프로토콜

```rust
// iroh-blobs/src/provider.rs

/// Blob 제공자
pub struct Provider {
    /// Endpoint
    endpoint: Endpoint,

    /// 로컬 저장소
    store: Store,
}

impl Provider {
    /// Provider 시작 (연결 수락)
    pub async fn run(self) -> Result<(), Error> {
        loop {
            // 1. Incoming 연결 수락
            let connecting = self.endpoint.accept().await?;
            let connection = connecting.await?;

            // 2. ALPN 확인
            if connection.alpn() != Some(b"iroh-blobs/1") {
                continue;
            }

            // 3. 핸들러 spawn
            let store = self.store.clone();
            tokio::spawn(async move {
                Self::handle_connection(connection, store).await
            });
        }
    }

    /// 연결 핸들러
    async fn handle_connection(connection: Connection, store: Store) -> Result<(), Error> {
        loop {
            // 1. Incoming 스트림 수락
            let (mut send, mut recv) = match connection.accept_bi().await {
                Ok(s) => s,
                Err(_) => break,  // 연결 종료
            };

            // 2. GET 요청 수신
            let mut buf = Vec::new();
            recv.read_to_end(&mut buf).await?;
            let request: GetRequest = bincode::deserialize(&buf)?;

            // 3. Blob 읽기
            let reader = store.read_blob(request.hash).await?;

            // 4. 응답 전송
            let response = GetResponse {
                size: reader.size(),
                data: (), // 실제로는 스트림
            };
            send.write_all(&bincode::serialize(&response)?).await?;

            // 5. 데이터 스트리밍
            let mut buf = vec![0u8; 64 * 1024];
            let mut total_sent = 0u64;

            loop {
                let n = reader.read(&mut buf).await?;
                if n == 0 {
                    break;
                }

                send.write_all(&buf[..n]).await?;
                total_sent += n as u64;
            }

            send.finish().await?;
            println!("Sent blob: {} ({} bytes)", request.hash, total_sent);
        }

        Ok(())
    }
}
```

---

## 6. iroh-gossip - Pub/Sub 오버레이

### 6.1 Gossip 프로토콜 개요

**위치**: `iroh-gossip/src/proto.rs`

iroh-gossip는 libp2p의 Gossipsub과 유사한 pub/sub 프로토콜입니다.

```rust
// iroh-gossip/src/proto.rs

/// Gossip 토픽 (32바이트)
pub type TopicId = [u8; 32];

/// Gossip 메시지
#[derive(Debug, Clone)]
pub struct Message {
    /// 발행자 NodeId
    pub from: NodeId,

    /// 토픽
    pub topic: TopicId,

    /// 페이로드
    pub data: Bytes,

    /// 시퀀스 번호 (중복 검출용)
    pub seq: u64,

    /// 서명 (발행자 검증)
    pub signature: Signature,
}

/// Gossip 네트워크
pub struct GossipNet {
    /// 로컬 NodeId
    node_id: NodeId,

    /// 구독 중인 토픽들
    subscriptions: HashSet<TopicId>,

    /// 피어 메시 (토픽별)
    ///
    /// TopicId -> Vec<NodeId>
    /// 각 토픽마다 6-12개 피어와 연결
    mesh: HashMap<TopicId, Vec<NodeId>>,

    /// 메시지 캐시 (중복 검출)
    ///
    /// (TopicId, SeqNo) -> Message
    /// TTL: 120초
    message_cache: LruCache<(TopicId, u64), Message>,

    /// Endpoint
    endpoint: Endpoint,
}

impl GossipNet {
    /// 토픽 구독
    pub async fn subscribe(&mut self, topic: TopicId) -> Result<(), Error> {
        // 1. 구독 추가
        self.subscriptions.insert(topic);

        // 2. 메시 구성 (6-12개 피어)
        let peers = self.find_peers_for_topic(topic).await?;
        self.mesh.insert(topic, peers.clone());

        // 3. 피어들에게 SUBSCRIBE 메시지 전송
        for peer in peers {
            self.send_control_message(peer, ControlMessage::Subscribe(topic)).await?;
        }

        Ok(())
    }

    /// 메시지 발행
    pub async fn publish(&mut self, topic: TopicId, data: Bytes) -> Result<(), Error> {
        // 1. 메시지 생성
        let seq = self.next_seq();
        let msg = Message {
            from: self.node_id,
            topic,
            data: data.clone(),
            seq,
            signature: self.sign_message(&topic, &data, seq),
        };

        // 2. 메시 피어들에게 전송
        let peers = self.mesh.get(&topic).ok_or(Error::NotSubscribed)?;

        for peer in peers {
            self.send_message(*peer, msg.clone()).await?;
        }

        // 3. 로컬 캐시에 저장
        self.message_cache.insert((topic, seq), msg.clone());

        Ok(())
    }

    /// 메시지 수신 핸들러
    async fn handle_message(&mut self, msg: Message) -> Result<(), Error> {
        // 1. 서명 검증
        if !self.verify_signature(&msg) {
            return Err(Error::InvalidSignature);
        }

        // 2. 중복 검출
        if self.message_cache.contains_key(&(msg.topic, msg.seq)) {
            return Ok(());  // 이미 본 메시지
        }

        // 3. 캐시에 저장
        self.message_cache.insert((msg.topic, msg.seq), msg.clone());

        // 4. 구독 중이면 애플리케이션에 전달
        if self.subscriptions.contains(&msg.topic) {
            self.emit_event(GossipEvent::Message(msg.clone()));
        }

        // 5. Gossip: 다른 피어들에게 전파
        //    (발신자 제외, 3-6개 피어에게)
        let peers_to_forward = self.select_gossip_peers(&msg.topic, &msg.from, 3..6);

        for peer in peers_to_forward {
            self.send_message(peer, msg.clone()).await?;
        }

        Ok(())
    }

    /// Heartbeat (주기적 메시 유지보수)
    ///
    /// 주기: 1초
    async fn heartbeat(&mut self) {
        for (topic, peers) in &mut self.mesh {
            // 1. 죽은 피어 제거
            peers.retain(|p| self.is_peer_alive(*p));

            // 2. 피어 수 부족하면 추가 (목표: 6-12개)
            while peers.len() < 6 {
                if let Some(new_peer) = self.find_new_peer_for_topic(*topic).await {
                    peers.push(new_peer);
                    self.send_control_message(new_peer, ControlMessage::Graft(*topic)).await;
                } else {
                    break;
                }
            }

            // 3. 피어 수 과다하면 제거
            while peers.len() > 12 {
                let removed = peers.pop().unwrap();
                self.send_control_message(removed, ControlMessage::Prune(*topic)).await;
            }
        }
    }
}
```

**Gossip 메시 토폴로지**:

```
토픽 "chat-room" 구독자들:

    A --- B --- C
    |  \  |  /  |
    |   \ | /   |
    D --- E --- F

각 노드는 6-12개 피어와 연결 (메시)
메시지는 모든 메시 연결로 전파

Gossip 파라미터:
- D (degree): 6-12 (메시 크기)
- Dhigh: 12 (최대)
- Dlow: 6 (최소)
- Heartbeat: 1초
- History: 120초 (중복 검출)
```

---

## 7. iroh-docs - CRDTs 문서 동기화

### 7.1 CRDT 개요

**위치**: `iroh-docs/src/engine.rs`

iroh-docs는 **Conflict-free Replicated Data Type (CRDT)**으로 분산 문서 동기화를 구현합니다.

```rust
// iroh-docs/src/engine.rs

use automerge::Automerge;  // CRDT 라이브러리

/// 문서 ID (32바이트)
pub type DocId = [u8; 32];

/// 문서 엔진
pub struct DocEngine {
    /// 로컬 문서들
    ///
    /// DocId -> Automerge document
    documents: HashMap<DocId, Automerge>,

    /// 동기화 상태
    ///
    /// (DocId, NodeId) -> SyncState
    sync_states: HashMap<(DocId, NodeId), SyncState>,

    /// Endpoint
    endpoint: Endpoint,
}

/// 동기화 상태
struct SyncState {
    /// 상대방이 가진 마지막 버전
    their_heads: Vec<ChangeHash>,

    /// 우리가 보낸 마지막 버전
    last_sent_heads: Vec<ChangeHash>,
}

impl DocEngine {
    /// 문서 생성
    pub fn create_doc(&mut self) -> DocId {
        let doc_id = DocId::random();
        let doc = Automerge::new();

        self.documents.insert(doc_id, doc);
        doc_id
    }

    /// 문서 편집
    ///
    /// Automerge CRDT:
    /// - 모든 변경은 operation으로 기록
    /// - 자동 병합 (conflict-free)
    pub fn edit_doc(&mut self, doc_id: DocId, key: &str, value: &str) -> Result<(), Error> {
        let doc = self.documents.get_mut(&doc_id).ok_or(Error::DocNotFound)?;

        // CRDT operation 생성
        let mut tx = doc.transaction();
        tx.put(automerge::ROOT, key, value)?;
        tx.commit();

        Ok(())
    }

    /// 피어와 동기화
    ///
    /// Automerge 동기화 프로토콜:
    /// 1. 각자의 heads 교환
    /// 2. 차이 계산
    /// 3. 누락된 changes 전송
    /// 4. 자동 병합
    pub async fn sync_with_peer(&mut self, doc_id: DocId, peer: NodeId) -> Result<(), Error> {
        // 1. 연결
        let connection = self.endpoint.connect(peer).await?;
        let (mut send, mut recv) = connection.open_bi().await?;

        // 2. 우리의 heads 전송
        let doc = self.documents.get(&doc_id).ok_or(Error::DocNotFound)?;
        let our_heads = doc.get_heads();

        send.write_all(&bincode::serialize(&SyncMessage::Heads(our_heads.clone()))?).await?;

        // 3. 상대방의 heads 수신
        let mut buf = Vec::new();
        recv.read_to_end(&mut buf).await?;
        let msg: SyncMessage = bincode::deserialize(&buf)?;

        let their_heads = match msg {
            SyncMessage::Heads(heads) => heads,
            _ => return Err(Error::ProtocolError),
        };

        // 4. 누락된 changes 계산
        let changes_to_send = doc.get_changes_since(&their_heads)?;

        // 5. Changes 전송
        if !changes_to_send.is_empty() {
            send.write_all(&bincode::serialize(&SyncMessage::Changes(changes_to_send))?).await?;
        }

        // 6. Changes 수신
        let mut buf = Vec::new();
        recv.read_to_end(&mut buf).await?;
        let msg: SyncMessage = bincode::deserialize(&buf)?;

        let their_changes = match msg {
            SyncMessage::Changes(changes) => changes,
            _ => return Err(Error::ProtocolError),
        };

        // 7. 자동 병합 (CRDT의 핵심!)
        //    Conflict 없이 병합됨
        if !their_changes.is_empty() {
            doc.apply_changes(their_changes)?;
        }

        println!("Synced doc {} with peer {}", doc_id, peer);

        Ok(())
    }
}
```

**CRDT 동작 예시**:

```
시나리오: 두 피어가 동시에 문서 편집

초기 상태 (양쪽 동일):
{ "title": "Hello" }

Peer A의 편집:
{ "title": "Hello", "author": "Alice" }

Peer B의 편집:
{ "title": "Hello", "count": 42 }

동기화 후 (자동 병합):
{ "title": "Hello", "author": "Alice", "count": 42 }

Conflict 발생 케이스 (같은 키):
Peer A: { "title": "Hello from A" }
Peer B: { "title": "Hello from B" }

Automerge 해결:
{ "title": "Hello from B" }  // Last-write-wins (타임스탬프 기반)
또는 두 값 모두 유지 (멀티 value)
```

---

## 8. 보안 모델 - PublicKey 기반 인증

### 8.1 Ed25519 키쌍

**위치**: `iroh/src/keys.rs`

```rust
// iroh/src/keys.rs

use ed25519_dalek::{Keypair, PublicKey, SecretKey, Signature};

/// Iroh의 키쌍 (Ed25519)
///
/// 용도:
/// 1. NodeId (공개키 = 주소)
/// 2. TLS 인증서 (QUIC 연결)
/// 3. 메시지 서명 (Gossip, Docs)
pub struct IrohKeypair {
    /// Secret key (32바이트)
    secret: SecretKey,

    /// Public key (32바이트)
    public: PublicKey,
}

impl IrohKeypair {
    /// 새 키쌍 생성 (랜덤)
    pub fn generate() -> Self {
        let keypair = Keypair::generate(&mut rand::rngs::OsRng);

        IrohKeypair {
            secret: keypair.secret,
            public: keypair.public,
        }
    }

    /// NodeId 반환 (공개키)
    pub fn node_id(&self) -> NodeId {
        NodeId(self.public.to_bytes())
    }

    /// 메시지 서명
    pub fn sign(&self, message: &[u8]) -> Signature {
        let keypair = Keypair {
            secret: self.secret,
            public: self.public,
        };

        keypair.sign(message)
    }

    /// 서명 검증 (static method)
    pub fn verify(public: &PublicKey, message: &[u8], signature: &Signature) -> bool {
        public.verify(message, signature).is_ok()
    }
}
```

### 8.2 보안 위협 모델

| 위협 | Iroh의 방어 |
|------|-------------|
| **중간자 공격 (MITM)** | QUIC TLS 1.3 + 공개키 검증 |
| **재생 공격** | QUIC nonce + 시퀀스 번호 |
| **Sybil 공격** | NodeId = 공개키 (위조 불가) |
| **Eclipse 공격** | 다중 relay + direct 연결 |
| **데이터 변조** | BLAKE3 verified streaming |

---

## 9. 실용 예제 코드

### 9.1 기본 Endpoint 사용

```rust
// examples/basic_endpoint.rs

use iroh::Endpoint;
use anyhow::Result;

#[tokio::main]
async fn main() -> Result<()> {
    // 1. Endpoint 생성
    let endpoint = Endpoint::bind().await?;

    println!("NodeId: {}", endpoint.node_id());
    println!("Local addr: {}", endpoint.local_addr());

    // 2. Incoming 연결 수락 루프
    tokio::spawn(async move {
        while let Ok(connecting) = endpoint.accept().await {
            tokio::spawn(handle_connection(connecting));
        }
    });

    // 프로그램 유지
    tokio::signal::ctrl_c().await?;

    Ok(())
}

async fn handle_connection(connecting: iroh::Connecting) -> Result<()> {
    let connection = connecting.await?;
    println!("Connected from: {}", connection.remote_node_id());

    // 양방향 스트림 수락
    let (mut send, mut recv) = connection.accept_bi().await?;

    // Echo 서버
    let mut buf = vec![0u8; 1024];
    loop {
        let n = recv.read(&mut buf).await?;
        if n == 0 {
            break;
        }

        send.write_all(&buf[..n]).await?;
    }

    Ok(())
}
```

### 9.2 Blob 공유 (Provider)

```rust
// examples/blob_provider.rs

use iroh::{Endpoint, protocol::Router};
use iroh_blobs::{BlobsProtocol, store::mem::MemStore};
use anyhow::Result;

#[tokio::main]
async fn main() -> Result<()> {
    // 1. Endpoint 생성
    let endpoint = Endpoint::builder()
        .discovery_n0()  // n0 DNS discovery 활성화
        .bind()
        .await?;

    println!("NodeId: {}", endpoint.node_id());

    // 2. Blob 저장소 생성
    let store = MemStore::new();

    // 3. BlobsProtocol 생성
    let blobs = BlobsProtocol::new(&store, endpoint.clone(), None);

    // 4. 프로토콜 라우터에 등록
    let router = Router::builder(endpoint)
        .accept(iroh_blobs::ALPN, blobs.clone())
        .spawn();

    // 5. Blob 추가
    let data = b"Hello, Iroh!";
    let tag = blobs.add_slice(data).await?;
    let ticket = blobs.ticket(tag).await?;

    println!("Blob ticket: {}", ticket);
    println!("Share this ticket to download the blob!");

    // 6. 서버 실행
    tokio::signal::ctrl_c().await?;
    router.shutdown().await?;

    Ok(())
}
```

### 9.3 Blob 다운로드 (Downloader)

```rust
// examples/blob_downloader.rs

use iroh::{Endpoint, protocol::Router};
use iroh_blobs::{BlobsProtocol, store::mem::MemStore, Ticket};
use anyhow::Result;

#[tokio::main]
async fn main() -> Result<()> {
    // Ticket from provider
    let ticket_str = std::env::args().nth(1).expect("Usage: blob_downloader <ticket>");
    let ticket: Ticket = ticket_str.parse()?;

    // 1. Endpoint 생성
    let endpoint = Endpoint::bind().await?;

    // 2. Blob 저장소
    let store = MemStore::new();

    // 3. BlobsProtocol
    let blobs = BlobsProtocol::new(&store, endpoint.clone(), None);

    // 4. 다운로드
    println!("Downloading blob...");
    let hash = ticket.hash();

    blobs.download_from_nodes(hash, ticket.nodes()).await?;

    // 5. 데이터 읽기
    let data = store.read_to_vec(hash).await?;
    println!("Downloaded: {}", String::from_utf8_lossy(&data));

    Ok(())
}
```

### 9.4 Gossip Pub/Sub

```rust
// examples/gossip_chat.rs

use iroh::{Endpoint, protocol::Router};
use iroh_gossip::{GossipProtocol, TopicId};
use anyhow::Result;

#[tokio::main]
async fn main() -> Result<()> {
    // 1. Endpoint
    let endpoint = Endpoint::bind().await?;
    let node_id = endpoint.node_id();

    // 2. Gossip 프로토콜
    let gossip = GossipProtocol::new(node_id);

    // 3. 라우터
    let router = Router::builder(endpoint)
        .accept(iroh_gossip::ALPN, gossip.clone())
        .spawn();

    // 4. 토픽 구독
    let topic = TopicId::from_str("chat-room")?;
    gossip.subscribe(topic).await?;

    println!("Subscribed to chat-room");

    // 5. 메시지 수신 루프
    let gossip_clone = gossip.clone();
    tokio::spawn(async move {
        while let Some(event) = gossip_clone.next_event().await {
            match event {
                GossipEvent::Message(msg) => {
                    let text = String::from_utf8_lossy(&msg.data);
                    println!("[{}] {}", msg.from, text);
                }
            }
        }
    });

    // 6. 메시지 발행 루프
    let mut interval = tokio::time::interval(std::time::Duration::from_secs(5));
    loop {
        interval.tick().await;

        let msg = format!("Hello from {}", node_id);
        gossip.publish(topic, msg.as_bytes().to_vec()).await?;
    }
}
```

---

## 10. 성능 최적화

### 10.1 QUIC 설정 튜닝

```rust
use iroh_quinn as quinn;

// QUIC transport 설정
let mut transport_config = quinn::TransportConfig::default();

// 1. 스트림 제한
transport_config.max_concurrent_bidi_streams(100u32.into());  // 양방향 스트림
transport_config.max_concurrent_uni_streams(100u32.into());   // 단방향 스트림

// 2. 흐름 제어
transport_config.stream_receive_window(1_000_000u32.into());  // 1MB per stream
transport_config.receive_window(10_000_000u32.into());        // 10MB total

// 3. Keep-alive
transport_config.keep_alive_interval(Some(Duration::from_secs(5)));

// 4. MTU
transport_config.mtu_discovery_config(Some(quinn::MtuDiscoveryConfig::default()));
```

### 10.2 Blob 전송 최적화

```rust
// 병렬 청크 다운로드
const CHUNK_SIZE: usize = 1024 * 1024;  // 1MB
const PARALLEL_CHUNKS: usize = 4;

for chunk_start in (0..total_size).step_by(CHUNK_SIZE).take(PARALLEL_CHUNKS) {
    let range = chunk_start..(chunk_start + CHUNK_SIZE).min(total_size);

    tokio::spawn(async move {
        download_range(hash, range).await
    });
}
```

---

## 11. Iroh vs libp2p 상세 비교

| 레이어 | libp2p | Iroh |
|--------|--------|------|
| **Transport** | TCP, QUIC, WS (선택) | QUIC 전용 |
| **Muxing** | Yamux, Mplex (별도) | QUIC 내장 |
| **Security** | Noise (업그레이드) | QUIC TLS 1.3 |
| **NAT 트래버설** | 수동 relay 설정 | 자동 (relay + hole punching) |
| **Protocol 협상** | Multistream-select | ALPN (TLS 확장) |
| **DHT** | Kademlia (복잡) | 선택적 (n0 DNS) |
| **Pub/Sub** | Gossipsub (많은 설정) | iroh-gossip (간단) |
| **Content Addressing** | 없음 (직접 구현) | iroh-blobs (BLAKE3) |
| **문서 동기화** | 없음 | iroh-docs (CRDT) |
| **API 복잡도** | ★★★★☆ | ★★☆☆☆ |
| **러닝 커브** | 가파름 (trait 이해 필요) | 완만 (간단한 API) |
| **생태계** | 크고 성숙 | 작지만 성장 중 |

---

## 12. 학습 로드맵

### 12.1 소스 코드 읽기 순서

```
Week 1: 기초
- iroh/src/endpoint.rs (Endpoint API)
- iroh-quinn/src/endpoint.rs (QUIC 기본)
- iroh/src/keys.rs (Ed25519 키쌍)

Week 2: NAT 트래버설
- iroh-net/src/magicsock.rs (MagicSocket)
- iroh-net/src/disco.rs (Discovery)
- iroh-relay/src/client.rs, server.rs (Relay 프로토콜)

Week 3: Blob 프로토콜
- iroh-blobs/src/protocol.rs (Blob 전송)
- iroh-blobs/src/store/ (저장소)
- iroh-blobs/src/downloader.rs (다운로드)

Week 4: 고급 프로토콜
- iroh-gossip/src/proto.rs (Gossip)
- iroh-docs/src/engine.rs (CRDT 동기화)
```

### 12.2 실습 프로젝트

1. **Week 1-2**: Echo 서버/클라이언트 (QUIC 기본)
2. **Week 3-4**: 파일 공유 앱 (iroh-blobs)
3. **Week 5-6**: 채팅 앱 (iroh-gossip)
4. **Week 7-8**: 협업 노트 (iroh-docs)

---

## 요약

Iroh는 **간단하고 작동하는 P2P**를 목표로 설계된 라이브러리입니다.

**핵심 특징**:
1. **QUIC 전용**: 빠른 연결, 내장 암호화/멀티플렉싱
2. **자동 NAT 트래버설**: Relay + Hole punching 자동 처리
3. **Ed25519 NodeId**: 32바이트 공개키가 곧 주소
4. **BLAKE3 컨텐츠 주소**: Verified streaming으로 무결성 보장
5. **간단한 API**: libp2p보다 훨씬 쉬움

**실전 사용**:
- 파일 공유 (iroh-blobs)
- 실시간 메시징 (iroh-gossip)
- 협업 문서 (iroh-docs)
- 커스텀 P2P 앱

이 문서로 Iroh의 전체 내부 구조를 100% 이해하고, 빠르게 P2P 애플리케이션을 구축할 수 있습니다.
