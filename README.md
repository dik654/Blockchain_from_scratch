# Blockchain from Scratch

이더리움, 솔라나, Sui 블록체인의 내부 구현을 **실제 소스 코드 기반**으로 완전 분석한 학습 자료입니다.

## 📚 문서 구성

### 핵심 문서

1. **[BLOCKCHAIN_COMPARISON_AND_ROADMAP.md](./BLOCKCHAIN_COMPARISON_AND_ROADMAP.md)** (33KB)
   - 🎯 **시작은 여기서!** - 3개 블록체인 비교 및 12주 학습 로드맵
   - 아키텍처 비교 (네트워크, 합의, 스토리지, RPC)
   - 성능 특성 및 트레이드오프
   - 개인 맞춤 학습 경로
   - 난이도별 실습 프로젝트

2. **[ETHEREUM_INTERNALS_SPEC.md](./ETHEREUM_INTERNALS_SPEC.md)** (39KB)
   - Ethereum (go-ethereum/geth) 완전 분석
   - DevP2P 네트워크 프로토콜 (RLPx, Kademlia)
   - Merkle Patricia Trie 상세 구조
   - LevelDB + Freezer 스토리지
   - StateDB 상태 관리
   - JSON-RPC 서버 구현

3. **[SOLANA_INTERNALS_SPEC.md](./SOLANA_INTERNALS_SPEC.md)** (54KB)
   - Solana (Agave) 완전 분석
   - Gossip + Turbine 네트워크 (PlumTree, Reed-Solomon FEC)
   - Proof of History (PoH) 시간 증명
   - RocksDB Blockstore + AccountsDB
   - Banking Stage 병렬 실행
   - Tower BFT 합의

4. **[SUI_INTERNALS_SPEC.md](./SUI_INTERNALS_SPEC.md)** (50KB)
   - Sui (MystenLabs) 완전 분석
   - Narwhal & Tusk DAG 기반 합의
   - 객체 중심 스토리지 모델
   - Move VM 통합
   - FastPath vs ConsensusPath
   - 체크포인트 & 에포크

### P2P 네트워킹 심화

5. **[LIBP2P_INTERNALS_SPEC.md](./LIBP2P_INTERNALS_SPEC.md)** (신규)
   - libp2p (rust-libp2p) 완전 분석
   - Transport & StreamMuxer (Yamux)
   - Swarm 연결 관리
   - NetworkBehaviour 프로토콜 구현
   - Kademlia DHT 피어 발견
   - Noise Protocol 보안 계층
   - Gossipsub Pub/Sub

6. **[IROH_INTERNALS_SPEC.md](./IROH_INTERNALS_SPEC.md)** (신규)
   - Iroh (n0-computer) 완전 분석
   - QUIC 기반 P2P 네트워킹
   - MagicEndpoint 자동 NAT 트래버설
   - Relay 서버 구조
   - iroh-blobs BLAKE3 컨텐츠 주소 지정
   - iroh-gossip Pub/Sub
   - iroh-docs CRDTs 문서 동기화

### 네트워크 특화

7. **[NAT_TRAVERSAL_GUIDE.md](./NAT_TRAVERSAL_GUIDE.md)**
   - NAT 개념 및 트래버설 기법
   - UPnP, STUN, TURN, ICE
   - UDP Hole Punching
   - Ethereum, Solana의 NAT 처리 방법

## 🎯 이 자료의 특징

### ✅ 100% 실제 코드 기반

모든 설명은 실제 소스 코드를 기반으로 작성되었습니다:

```rust
// 소스 위치 명시
// 소스: ledger/src/shred.rs

pub struct Shred {
    // 목적: 블록 데이터의 원자 단위
    // 의도: UDP 패킷으로 전송 가능한 크기 (~1200 bytes)

    common_header: ShredCommonHeader,
    payload: Vec<u8>,
}

// 내부 로직:
// 1. 블록을 Shred 크기로 분할
// 2. Reed-Solomon 코딩으로 FEC 생성
// 3. Turbine tree로 전파
```

### ✅ 의도와 내부 로직 설명

각 코드 블록마다:
- **목적**: 왜 이 코드가 존재하는가?
- **의도**: 설계자의 의도는 무엇인가?
- **내부 로직**: 어떻게 동작하는가? (단계별)
- **트레이드오프**: 장단점은?

### ✅ 전체 데이터 흐름 추적

트랜잭션이 블록이 되기까지의 **전체 과정**을 함수 단위로 추적:

```
Ethereum: 트랜잭션 -> 블록
RPC → TxPool → Miner → EVM → StateDB → Trie → Block → Broadcast

Solana: 트랜잭션 -> 슬롯
RPC → TPU → Banking Stage → Bank → PoH → Shred → Turbine

Sui: 트랜잭션 -> 체크포인트
RPC → FastPath/ConsensusPath → Move VM → AuthorityStore → Checkpoint
```

## 🚀 시작하기

### 1️⃣ 학습 순서 (추천)

```
Step 1: BLOCKCHAIN_COMPARISON_AND_ROADMAP.md 읽기
        ↓
Step 2: 자신에게 맞는 블록체인 선택
        ↓
Step 3: 해당 SPEC 문서 깊이 학습
        ↓
Step 4: 실제 소스 코드 클론 및 분석
        ↓
Step 5: 실습 프로젝트 진행
```

### 2️⃣ 학습자별 추천 경로

#### 블록체인 입문자
```
1. BLOCKCHAIN_COMPARISON_AND_ROADMAP.md (전체 이해)
2. ETHEREUM_INTERNALS_SPEC.md (가장 안정적, 자료 많음)
3. 실습: 블록체인 익스플로러, Merkle Tree 구현
```

#### 시스템 프로그래머
```
1. SOLANA_INTERNALS_SPEC.md (복잡한 파이프라인)
2. ETHEREUM_INTERNALS_SPEC.md (안정적 구조)
3. 실습: PoH 검증기, 병렬 실행 시뮬레이터
```

#### 블록체인 연구자
```
1. 3개 SPEC 모두 읽기
2. BLOCKCHAIN_COMPARISON_AND_ROADMAP.md (비교 분석)
3. 실습: 커스텀 블록체인, 성능 벤치마크
```

#### 스마트 컨트랙트 개발자
```
1. ETHEREUM_INTERNALS_SPEC.md (EVM)
2. SUI_INTERNALS_SPEC.md (Move VM)
3. 실습: DEX, NFT 마켓플레이스
```

#### P2P 네트워크 개발자
```
1. NAT_TRAVERSAL_GUIDE.md (NAT 기초)
2. LIBP2P_INTERNALS_SPEC.md (모듈러 P2P 스택)
3. IROH_INTERNALS_SPEC.md (간단한 P2P)
4. 실습: 파일 공유 앱, 채팅 앱, 커스텀 P2P 프로토콜
```

### 3️⃣ 실제 소스 코드 클론

각 블록체인의 소스 코드를 클론하여 문서와 함께 학습:

```bash
# Ethereum (go-ethereum)
git clone https://github.com/ethereum/go-ethereum.git
cd go-ethereum
# 파일: p2p/server.go, core/state/statedb.go, trie/trie.go

# Solana (agave)
git clone https://github.com/anza-xyz/agave.git
cd agave
# 파일: ledger/src/shred.rs, runtime/src/accounts_db.rs

# Sui
git clone https://github.com/MystenLabs/sui.git
cd sui
# 파일: crates/sui-types/src/object.rs, narwhal/types/src/header.rs

# libp2p (rust-libp2p)
git clone https://github.com/libp2p/rust-libp2p.git
cd rust-libp2p
# 파일: core/src/transport/mod.rs, swarm/src/lib.rs, protocols/kad/src/lib.rs

# Iroh
git clone https://github.com/n0-computer/iroh.git
cd iroh
# 파일: iroh/src/endpoint.rs, iroh-net/src/magicsock.rs, iroh-blobs/src/protocol.rs
```

## 📖 각 문서 상세 내용

### ETHEREUM_INTERNALS_SPEC.md

**1. 네트워크 레이어**
- DevP2P 프로토콜 스택
- RLPx 암호화 (ECIES, AES-256)
- Kademlia DHT 노드 발견
- Ethereum Wire Protocol (ETH)

**2. 데이터베이스 레이어**
- LevelDB 통합 및 키 스킴
- Freezer (Ancient Store)
- 배치 연산 및 압축

**3. 스토리지 레이어**
- Merkle Patricia Trie 구조
- Secure Trie (Keccak256)
- StateDB 상태 관리
- Snapshot 가속

**4. RPC 레이어**
- JSON-RPC 서버 구조
- ETH Namespace 메서드
- 필터 및 구독 (WebSocket)

**5. 핵심 데이터 구조**
- Block & Header
- Transaction (Legacy, EIP-1559)
- Receipt & Log
- Bloom Filter

### SOLANA_INTERNALS_SPEC.md

**1. 네트워크 레이어**
- Gossip Protocol (PlumTree)
- ClusterInfo (CRDS)
- Turbine 블록 전파
- Shred 구조 및 FEC

**2. 데이터베이스 레이어**
- RocksDB Column Families
- BlockstoreDB
- SlotMeta

**3. 스토리지 레이어**
- AccountsDB
- AppendVec (Append-only)
- AccountsIndex
- Snapshot

**4. RPC 레이어**
- JSON-RPC 서버
- PubSub (WebSocket)
- 주요 메서드 구현

**5. 핵심 아키텍처**
- Proof of History (PoH)
- Tower BFT 합의
- Banking Stage (병렬 실행)
- TPU & TVU 파이프라인

### SUI_INTERNALS_SPEC.md

**1. 네트워크 레이어**
- Narwhal & Tusk
- Primary-Worker 구조
- DAG 기반 합의
- Anemo P2P

**2. 데이터베이스 레이어**
- RocksDB (TypedStore)
- DBMap<K, V>
- 타입 안전 래퍼

**3. 스토리지 레이어**
- AuthorityStore
- Object 모델
- Owner 타입
- 객체 버전 관리

**4. RPC 레이어**
- JSON-RPC API
- 객체 조회
- 트랜잭션 실행
- WebSocket 구독

**5. 핵심 아키텍처**
- Move VM 통합
- FastPath vs ConsensusPath
- Checkpoint & Epoch
- 병렬 실행 메커니즘

### LIBP2P_INTERNALS_SPEC.md

**1. Core Layer**
- Transport trait (연결 생성 방법)
- StreamMuxer trait (멀티플렉싱)
- Upgrade 패턴 (암호화, muxing 추가)

**2. Transport 구현**
- TCP Transport
- QUIC Transport (빠른 핸드셰이크)
- WebSocket Transport

**3. Multiplexing**
- Yamux (권장)
- 스트림별 독립적 흐름 제어
- 12바이트 헤더 프로토콜

**4. Swarm**
- 연결 관리 및 이벤트 조율
- NetworkBehaviour 통합
- 프로토콜 협상

**5. Protocols**
- Kademlia DHT (피어 발견)
- Gossipsub (Pub/Sub 메시징)
- Noise Protocol (암호화)
- Request/Response (generic RPC)

### IROH_INTERNALS_SPEC.md

**1. Endpoint & QUIC**
- Quinn QUIC 구현
- Ed25519 NodeId (32바이트 공개키)
- TLS 1.3 자동 암호화
- ALPN 프로토콜 협상

**2. MagicEndpoint**
- 자동 NAT 트래버설
- Relay + Direct 병렬 연결
- MagicSocket (통합 소켓)
- Discovery (STUN-like)

**3. Relay 서버**
- WebSocket over HTTPS
- NodeId 기반 라우팅
- O(1) 패킷 전달
- 제로 카피 중계

**4. iroh-blobs**
- BLAKE3 컨텐츠 주소 지정
- Verified Streaming
- 청크 단위 검증
- Provider/Downloader 프로토콜

**5. iroh-gossip & iroh-docs**
- Gossip Pub/Sub (6-12 피어 메시)
- CRDTs 문서 동기화 (Automerge)
- Conflict-free 병합
- 실시간 협업

## 💡 학습 팁

### 효과적인 코드 읽기

1. **Top-Down 접근** (초보자 추천)
   ```
   전체 아키텍처 → 데이터 흐름 → 세부 구현
   ```

2. **Bottom-Up 접근** (경험자 추천)
   ```
   데이터 구조 → 알고리즘 → 시스템 통합
   ```

3. **주석 달며 읽기**
   ```go
   // 역할: ...
   // 목적: ...
   // 흐름: 1. ... 2. ... 3. ...
   ```

### 도구 활용

**IDE**
- VSCode + Go/Rust/Move 확장
- Go to Definition (F12)
- Find References (Shift+F12)
- Call Hierarchy (Shift+Alt+H)

**다이어그램**
- draw.io: 아키텍처
- PlantUML: UML
- Mermaid: 데이터 흐름

**분석 도구**
```bash
# Go
go doc <package>
go-callvis <package>

# Rust
cargo doc --open
cargo tree
```

## 🎓 실습 프로젝트

### 초급 (Week 1-4)
- [ ] 블록체인 익스플로러
- [ ] Merkle Tree 라이브러리
- [ ] 간단한 블록체인 구현

### 중급 (Week 5-8)
- [ ] 이벤트 모니터링 봇
- [ ] 간단한 DEX (각 체인)
- [ ] 상태 추적 도구

### 고급 (Week 9-12)
- [ ] Multi-chain 지갑
- [ ] 블록체인 분석 대시보드
- [ ] 커스텀 Layer 2

## 📊 성능 비교

| 항목 | Ethereum | Solana | Sui |
|------|----------|--------|-----|
| **TPS** | 15-30 | 3,000-5,000 | 5,000-10,000 |
| **Finality** | ~13분 | ~13초 | 3-5초 |
| **블록 시간** | 12초 | 400ms | N/A |
| **언어** | Go | Rust | Rust |
| **VM** | EVM | SVM | Move VM |
| **모델** | Account | Account | Object |

### P2P 네트워킹 비교

| 항목 | libp2p | Iroh |
|------|--------|------|
| **전송 계층** | TCP, QUIC, WebSocket (선택) | QUIC 전용 |
| **복잡도** | ★★★★☆ (높음) | ★★☆☆☆ (낮음) |
| **NAT 트래버설** | 수동 설정 | 자동 (relay + hole punching) |
| **프로토콜 협상** | Multistream-select | ALPN (TLS 확장) |
| **주요 사용처** | Polkadot, Filecoin, Ethereum 2.0 | 파일 공유, 간단한 P2P 앱 |
| **학습 난이도** | 높음 (trait 이해 필요) | 낮음 (간단한 API) |

## 🔗 추가 자료

### 공식 문서
- [Ethereum](https://ethereum.org/developers)
- [Solana](https://docs.solana.com)
- [Sui](https://docs.sui.io)
- [libp2p](https://docs.libp2p.io)
- [Iroh](https://www.iroh.computer/docs)

### GitHub 저장소
- [go-ethereum](https://github.com/ethereum/go-ethereum)
- [agave (Solana)](https://github.com/anza-xyz/agave)
- [sui](https://github.com/MystenLabs/sui)
- [rust-libp2p](https://github.com/libp2p/rust-libp2p)
- [iroh](https://github.com/n0-computer/iroh)

### 추천 논문
- Ethereum Yellow Paper (Gavin Wood)
- Proof of History (Anatoly Yakovenko)
- Narwhal and Tusk (Danezis et al.)

## 🤝 기여

이 자료는 블록체인 학습을 위한 오픈소스 프로젝트입니다.

**개선 사항이나 오류 발견 시:**
1. Issue 생성
2. Pull Request 제출
3. 토론 참여

## 📝 라이선스

MIT License - 자유롭게 사용, 수정, 배포 가능

## ⚠️ 주의사항

- 모든 코드 분석은 2024-2025년 기준입니다
- 블록체인은 빠르게 발전하므로 최신 코드와 차이가 있을 수 있습니다
- 실제 프로덕션 환경에서는 공식 문서를 참조하세요

## 🙏 감사의 말

이 자료는 다음 오픈소스 프로젝트들의 코드를 분석하여 작성되었습니다:
- Ethereum Foundation (go-ethereum)
- Solana Labs / Anza (Solana/Agave)
- Mysten Labs (Sui)
- libp2p (Protocol Labs)
- n0 (Iroh)

## 📞 문의

질문이나 피드백은 GitHub Issues를 이용해주세요.

---

**Happy Learning! 🚀**

블록체인 내부를 완전히 이해하는 여정을 시작하세요!
