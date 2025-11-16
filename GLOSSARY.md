# 블록체인 용어 사전

블록체인 내부 구현을 학습하면서 만나는 주요 용어들의 정의와 설명입니다.

---

## A

### Account (계정)
**Ethereum/Solana**: 블록체인 상태의 기본 단위. 주소, 잔액, 논스 등을 포함.
- **EOA (Externally Owned Account)**: 개인키로 제어되는 계정
- **Contract Account**: 스마트 컨트랙트 코드를 포함하는 계정

**Sui**: 계정 대신 객체(Object) 모델 사용

### AccountsDB
**Solana**: 모든 계정 상태를 저장하는 데이터베이스
- AppendVec 파일로 구성
- 메모리 맵 파일 사용
- 병렬 접근 지원

**관련**: `runtime/src/accounts_db.rs`

### Anchor
**Solana Anchor**: Solana 스마트 컨트랙트 개발 프레임워크
**Sui Narwhal**: DAG에서 순서 결정의 기준점이 되는 인증서

### Append-only
데이터를 끝에만 추가하는 방식
- **장점**: 빠른 쓰기, 간단한 구조
- **단점**: 압축(Compaction) 필요
- **예시**: Solana AccountsDB, Sui ObjectStore

### AppendVec
**Solana**: Append-only 계정 저장 파일
- 고정 크기 (보통 64MB)
- 메모리 맵으로 접근
- 계정 데이터를 순차적으로 저장

**관련**: `runtime/src/append_vec.rs`

---

## B

### BFT (Byzantine Fault Tolerance)
비잔틴 장애 허용
- 악의적 노드가 있어도 합의 가능
- 일반적으로 2/3 이상의 정직한 노드 필요
- **예시**: PBFT, Tower BFT, Narwhal-Tusk

### Banking Stage
**Solana**: 트랜잭션을 배치로 실행하는 파이프라인 단계
- 계정 충돌 감지
- 병렬 배치 실행
- PoH에 기록

**관련**: `core/src/banking_stage.rs`

### Batch
트랜잭션들의 묶음
- **Solana**: Worker가 생성한 트랜잭션 배치
- **Ethereum**: 원자적으로 실행되는 상태 변경들
- **목적**: 효율성, 원자성 보장

### Blockchain
연결된 블록들의 체인
```
Genesis -> Block 1 -> Block 2 -> ... -> Latest Block
```
각 블록은 이전 블록의 해시를 포함하여 변조 방지

### Bloom Filter
확률적 데이터 구조
- **용도**: 빠른 포함 여부 검사
- **특징**: False positive 가능, False negative 불가능
- **Ethereum**: 로그 필터링에 사용

**예시**:
```go
// Ethereum Block Header의 Bloom
type Header struct {
    Bloom Bloom  // 2048 bits
    // ...
}
```

---

## C

### Casper FFG (Friendly Finality Gadget)
**Ethereum**: PoS 합의 프로토콜의 Finality 부분
- 체크포인트 시스템
- Justification과 Finalization
- 슬래싱 메커니즘

### Certificate
**Sui Narwhal**: Quorum 서명을 받은 Header
- 2/3+ 검증자 서명 필요
- DAG의 확정된 노드
- 순서 결정에 사용

**구조**:
```rust
struct Certificate {
    header: Header,
    aggregated_signature: AggregateSignature,
    signers: Vec<PublicKey>,
}
```

### Checkpoint
**Sui**: 일정 트랜잭션을 묶은 배치
- Finality 제공
- 동기화 단위
- 검증자 서명 포함

**Ethereum**: 합의 체크포인트 (32 슬롯마다)

### Commitment
데이터에 대한 암호학적 약속
- 나중에 공개 가능
- 변경 불가능
- **예시**: Merkle root, 블록 해시

### Compaction
**데이터베이스**: 오래된 데이터 정리 및 공간 회수
- **LevelDB/RocksDB**: SST 파일 병합
- **Solana AccountsDB**: 살아있는 계정만 재작성
- **목적**: 디스크 공간 절약, 성능 향상

### Consensus
분산 시스템에서 합의 도달
- **Ethereum**: Gasper (Casper FFG + LMD GHOST)
- **Solana**: Tower BFT + PoH
- **Sui**: Narwhal-Tusk / Mysticeti

### ConsensusPath
**Sui**: 공유 객체를 포함한 트랜잭션 경로
- Narwhal을 통한 합의 필요
- 순차 실행
- 레이턴시 증가

**vs FastPath**: 소유 객체만 사용하면 FastPath

### CRDS (Cluster Replicated Data Store)
**Solana**: Gossip 네트워크의 데이터 저장소
- CRDT (Conflict-free Replicated Data Type) 기반
- 결과적 일관성
- 벡터 클락으로 버전 관리

---

## D

### DAG (Directed Acyclic Graph)
방향성 비순환 그래프
- **Sui Narwhal**: 트랜잭션 DAG
- **장점**: 병렬 처리, 높은 처리량
- **단점**: 복잡성

### DevP2P
**Ethereum**: P2P 네트워킹 프로토콜 스택
- RLPx: 암호화 전송
- Discovery: 노드 발견
- ETH Protocol: 블록체인 데이터 교환

**관련**: `p2p/` 디렉토리

### DHT (Distributed Hash Table)
분산 해시 테이블
- **Ethereum**: Kademlia DHT로 노드 발견
- 키-값 쌍을 분산 저장
- O(log n) 조회

### Digest
해시값
- 데이터의 암호학적 요약
- 고정 크기 (예: 32 bytes)
- **용도**: 식별, 검증, 참조

---

## E

### ECDSA (Elliptic Curve Digital Signature Algorithm)
타원 곡선 디지털 서명
- **Ethereum**: secp256k1 곡선
- **Solana**: Ed25519 사용 (더 빠름)
- **용도**: 트랜잭션 서명, 계정 인증

### Ed25519
에드워드 곡선 서명 알고리즘
- **Solana**: 주로 사용
- **장점**: ECDSA보다 빠름, 작은 서명
- **단점**: 하드웨어 지원 적음

### ENR (Ethereum Node Records)
**Ethereum**: 노드 연결 정보의 서명된 레코드
- IP, 포트, 공개키 등
- Base64 URL 인코딩
- **형식**: `enr:-IS4Q...`

### Entry
**Solana PoH**: PoH 스트림의 단위
- Tick (시간) 또는
- 트랜잭션 (데이터)

```rust
struct Entry {
    hash: Hash,
    num_hashes: u64,
    transactions: Vec<Transaction>,
}
```

### Epoch
일정 기간 또는 블록 수
- **Ethereum**: 32 슬롯 (약 6.4분)
- **Solana**: 432,000 슬롯 (약 2일)
- **Sui**: 검증자 집합이 고정된 기간

### Erasure Coding
데이터 복구 기법
- **Solana Turbine**: Reed-Solomon 코딩
- 67 data shreds + 33 coding shreds
- 67개만 받아도 100개 복구 가능

### EVM (Ethereum Virtual Machine)
**Ethereum**: 스마트 컨트랙트 실행 환경
- 스택 기반 가상 머신
- Gas 미터링
- Deterministic 실행

---

## F

### FastPath
**Sui**: 소유 객체만 사용하는 트랜잭션 경로
- 합의 불필요
- 즉시 실행
- 병렬 처리 가능

**조건**: 모든 입력 객체가 AddressOwner 또는 ObjectOwner

### FEC (Forward Error Correction)
전진 오류 수정
- 데이터 손실 복구
- **Solana**: Reed-Solomon 코딩
- **용도**: Shred 복구

### Finality
트랜잭션이 되돌릴 수 없는 상태
- **Ethereum**: ~13분 (2 epochs)
- **Solana**: ~13초 (32 slots)
- **Sui**: 3-5초 (checkpoint)

### Freezer
**Ethereum**: 오래된 블록의 Append-only 저장소
- 압축 저장
- 순차 읽기 최적화
- **위치**: Ancient 데이터

**관련**: `core/rawdb/freezer.go`

---

## G

### Gas
**Ethereum**: 계산 비용 단위
- Gas price: 단위당 가격 (wei)
- Gas limit: 최대 Gas
- Gas used: 실제 사용량

**EIP-1559**:
- Base fee: 자동 조정
- Priority fee: 채굴자 팁

### Gasper
**Ethereum**: PoS 합의 프로토콜
= Casper FFG (Finality) + LMD GHOST (포크 선택)

### Gossip
P2P 정보 전파 프로토콜
- **Solana**: PlumTree 기반
- Push/Pull 메커니즘
- **용도**: 제어 평면 (메타데이터)

---

## H

### Hash
암호학적 해시 함수의 출력
- **특성**: 일방향, 충돌 저항성, Deterministic
- **Ethereum**: Keccak-256
- **Solana/Sui**: SHA-256, Blake2b

### Header
블록 헤더
- 블록 메타데이터
- 이전 블록 해시
- 상태 루트
- 트랜잭션 루트

**Ethereum**:
```go
type Header struct {
    ParentHash  common.Hash
    Root        common.Hash  // State root
    TxHash      common.Hash
    // ...
}
```

### HD Wallet (Hierarchical Deterministic Wallet)
계층적 결정론적 지갑
- 하나의 시드로 여러 키 파생
- **BIP-32/44**: 표준
- **용도**: 키 관리 단순화

---

## I

### Immutable
**Sui**: 변경 불가능한 객체
- 소유자 없음
- 영구적
- 읽기만 가능

**예시**: 패키지, 상수 객체

### Index
데이터베이스 인덱스
- 빠른 조회를 위한 자료구조
- **Ethereum**: Transaction lookup
- **Solana**: AccountsIndex
- **Sui**: Owner index

---

## J

### JSON-RPC
원격 프로시저 호출 프로토콜
- HTTP/WebSocket으로 전송
- **형식**: `{"jsonrpc": "2.0", "method": "...", "params": [...]}`
- **모든 체인**: 주요 API 프로토콜

---

## K

### Kademlia
분산 해시 테이블 알고리즘
- **Ethereum DevP2P**: 노드 발견
- XOR 거리 메트릭
- K-bucket (k=16)
- O(log n) 조회

### Keccak-256
**Ethereum**: 주요 해시 함수
- SHA-3 계열
- 256-bit 출력
- **용도**: 계정 주소, 트라이 키

---

## L

### Leader
**Solana**: 현재 블록을 생성하는 검증자
- PoH 기반 스케줄
- 에포크마다 결정
- 스테이크 비례 슬롯 할당

### LevelDB
키-값 저장소
- **Ethereum**: 메인 데이터베이스
- Google 개발
- LSM-tree 기반
- **특징**: 빠른 쓰기, 압축

### Lockout
**Solana Tower BFT**: 투표 잠금 메커니즘
- 거리 = 2^confirmation_count 슬롯
- 되돌리기 어렵게 만듦
- **목적**: Finality 보장

---

## M

### Mempool
미확인 트랜잭션 풀
- **Ethereum**: TxPool
- **Solana**: 각 검증자의 큐
- **Sui Narwhal**: DAG mempool (분리!)

### Merkle Patricia Trie
**Ethereum**: 상태 저장 자료구조
= Merkle tree + Patricia trie
- 경로 압축
- Merkle proof 지원
- **3개 트라이**: State, Storage, Transaction

### Merkle Proof
특정 데이터가 Merkle tree에 포함되었음을 증명
- O(log n) 크기
- 루트 해시로 검증
- **용도**: 경량 클라이언트

### Move
**Sui**: 스마트 컨트랙트 언어
- 리소스 중심
- Formal verification 지원
- **특징**: 안전성, 표현력

### Mysticeti
**Sui**: Narwhal-Tusk를 개선한 새 합의 프로토콜
- 더 낮은 레이턴시
- 간소화된 구조

---

## N

### Narwhal
**Sui**: DAG 기반 Mempool 프로토콜
- Primary-Worker 아키텍처
- 전파와 합의 분리
- 높은 처리량

### Nonce
일회용 번호
- **Ethereum Account**: 트랜잭션 순서 (replay 공격 방지)
- **PoW**: 채굴 논스
- **암호학**: 랜덤 값

---

## O

### Object
**Sui**: 블록체인 상태의 기본 단위
```rust
struct Object {
    id: UID,
    owner: Owner,
    data: Data,
    version: SequenceNumber,
}
```

**vs Account**: 명확한 소유권, 병렬 처리 가능

### Owner
**Sui**: 객체 소유권 타입
- **AddressOwner**: 주소 소유
- **Shared**: 공유
- **ObjectOwner**: 객체 소유
- **Immutable**: 불변

---

## P

### P2P (Peer-to-Peer)
중앙 서버 없는 네트워크
- 모든 노드가 동등
- **Ethereum**: DevP2P
- **Solana**: Gossip
- **Sui**: Anemo

### Patricia Trie
경로 압축 트라이
- 공통 접두사 압축
- 공간 효율적
- **Ethereum**: Merkle Patricia Trie

### PoH (Proof of History)
**Solana**: 시간의 암호학적 증명
- SHA-256 해시 체인
- 순차적으로만 계산 가능
- **목적**: 신뢰할 수 있는 시계

```
Hash_0 = SHA256(seed)
Hash_1 = SHA256(Hash_0)
Hash_n = SHA256(Hash_{n-1})
```

### PoS (Proof of Stake)
지분 증명
- **Ethereum**: Casper FFG + LMD GHOST
- 32 ETH 예치
- 슬래싱 메커니즘

### Primary
**Sui Narwhal**: 합의 참여 노드
- Header 생성
- Vote 수집
- Certificate 발행

### Prune
데이터 정리
- 오래된 객체 버전 삭제
- **Sui**: num-epochs-to-retain 설정
- **목적**: 디스크 공간 절약

---

## Q

### Quorum
정족수
- 합의에 필요한 최소 참여자
- 보통 2/3 이상 (BFT)
- **Sui Certificate**: 2/3+ 스테이크

---

## R

### Receipt
트랜잭션 실행 결과
- **Ethereum**: 상태, Gas 사용량, 로그
- **Solana**: Transaction metadata
- **Sui**: TransactionEffects

### Reed-Solomon
Erasure coding 알고리즘
- **Solana Turbine**: FEC
- (n, k) 코드: n개 중 k개로 복구
- **예**: (100, 67) = 67개로 100개 복구

### Replay Attack
같은 트랜잭션 재사용 공격
- **방어**: Nonce (Ethereum), Recent blockhash (Solana)

### RLP (Recursive Length Prefix)
**Ethereum**: 직렬화 형식
- 간단하고 결정론적
- **용도**: 블록, 트랜잭션, 상태 인코딩

### RLPx
**Ethereum DevP2P**: 암호화 전송 프로토콜
- ECIES 키 교환
- AES-256-CTR 암호화
- MAC으로 무결성

### RocksDB
키-값 저장소
- **Solana/Sui**: 메인 데이터베이스
- LevelDB 포크
- **특징**: 높은 성능, Column Families

---

## S

### Sealevel
**Solana**: 병렬 실행 런타임
- 계정 충돌 기반
- 수만 개 코어 지원 가능

### Shared Object
**Sui**: 공유 객체
- 누구나 접근 가능
- 합의 필요 (ConsensusPath)
- **예**: DEX pool, DAO 금고

### Shred
**Solana**: 블록의 조각
- ~1200 bytes (UDP 패킷)
- Data shred (실제 데이터)
- Coding shred (FEC)

### Slot
시간 단위
- **Ethereum**: 12초
- **Solana**: ~400ms
- **Sui**: 없음 (DAG)

### Snapshot
특정 시점의 상태 스냅샷
- **Ethereum**: State snapshot (평탄화)
- **Solana**: AccountsDB snapshot
- **Sui**: RocksDB checkpoint
- **목적**: 빠른 동기화

### Stake
지분
- PoS에서 검증자 권한
- **Ethereum**: 32 ETH
- **Solana**: 최소 없음
- **목적**: 보안, 인센티브

### State
블록체인의 현재 상태
- **Ethereum**: 모든 계정의 잔액, 스토리지
- **Solana**: 모든 계정 데이터
- **Sui**: 모든 객체

### StateDB
**Ethereum**: 상태 관리 데이터베이스
- Merkle Patricia Trie 래퍼
- 캐시 포함
- Copy-on-Write

---

## T

### Tick
**Solana PoH**: 시간 단위
- 고정 횟수 해시 (예: 1000번)
- ~6.25ms
- **목적**: 일정한 시간 간격

### Tower BFT
**Solana**: PoH 기반 합의
- 투표 타워 (스택)
- Lockout 메커니즘
- **목적**: Finality

### TPU (Transaction Processing Unit)
**Solana**: 리더의 트랜잭션 처리 파이프라인
- FetchStage
- SigVerifyStage
- BankingStage
- BroadcastStage

### Transaction
상태 변경 요청
- 서명 필수
- Gas/Fee 지불
- **Ethereum**: To, Value, Data, Nonce
- **Solana**: Instructions, Accounts
- **Sui**: ProgrammableTransaction

### Trie
트리 자료구조
- **Ethereum**: Merkle Patricia Trie
- 키-값 저장
- Merkle root 제공

### Turbine
**Solana**: 블록 전파 프로토콜
- Multi-layer tree
- Shred 기반
- **목적**: 빠른 전파 (<200ms)

### Tusk
**Sui Narwhal**: DAG 순서화 합의
- Zero-message overhead
- Anchor 기반
- **목적**: DAG → 선형 순서

### TVU (Transaction Validation Unit)
**Solana**: 검증자의 블록 검증 파이프라인
- Shred 수신
- 재전파
- 검증

---

## U

### UID
**Sui**: 객체 고유 식별자
- 32 bytes
- 글로벌하게 유일
- `object::new(ctx)`로 생성

### UTXO (Unspent Transaction Output)
**Bitcoin**: 미사용 트랜잭션 출력
- **참고**: Ethereum은 Account 모델

---

## V

### Validator
검증자
- 블록 생성/검증
- 합의 참여
- 스테이크 예치 필요

### Verifiable Delay Function (VDF)
검증 가능한 지연 함수
- 순차적으로만 계산 가능
- 빠른 검증
- **PoH**: VDF와 유사한 원리

### Version
**Sui Object**: 객체 버전
- SequenceNumber
- 매 업데이트마다 증가
- **용도**: 버전 히스토리, 충돌 감지

---

## W

### Wallet
지갑
- 개인키 관리
- 트랜잭션 서명
- **타입**: Hot, Cold, Hardware, HD

### Worker
**Sui Narwhal**: 배치 생성 노드
- 트랜잭션 수신
- 배치로 그룹화
- Primary에 전달

---

## X

### XOR Distance
**Kademlia**: 거리 메트릭
- 두 노드 ID의 XOR
- 대칭적, 삼각 부등식
- **용도**: 가까운 노드 찾기

---

## Z

### Zero-Knowledge Proof
영지식 증명
- 정보 공개 없이 증명
- **ZK-SNARK**: Succinct
- **ZK-STARK**: Transparent
- **용도**: 프라이버시, Layer 2

---

## 약어 정리

| 약어 | 의미 | 설명 |
|------|------|------|
| API | Application Programming Interface | 프로그램 인터페이스 |
| BFT | Byzantine Fault Tolerance | 비잔틴 장애 허용 |
| COW | Copy-on-Write | 쓰기시 복사 |
| CRDT | Conflict-free Replicated Data Type | 충돌 없는 복제 데이터 |
| DAG | Directed Acyclic Graph | 방향성 비순환 그래프 |
| DHT | Distributed Hash Table | 분산 해시 테이블 |
| ECDSA | Elliptic Curve Digital Signature Algorithm | 타원 곡선 서명 |
| ENR | Ethereum Node Records | 이더리움 노드 레코드 |
| EVM | Ethereum Virtual Machine | 이더리움 가상 머신 |
| FEC | Forward Error Correction | 전진 오류 수정 |
| HD | Hierarchical Deterministic | 계층적 결정론적 |
| JSON | JavaScript Object Notation | 데이터 형식 |
| LSM | Log-Structured Merge | 로그 구조 병합 |
| MAC | Message Authentication Code | 메시지 인증 코드 |
| MEV | Miner Extractable Value | 채굴자 추출 가치 |
| MPT | Merkle Patricia Trie | 머클 패트리샤 트라이 |
| P2P | Peer-to-Peer | 피어 투 피어 |
| PoH | Proof of History | 역사 증명 |
| PoS | Proof of Stake | 지분 증명 |
| PoW | Proof of Work | 작업 증명 |
| RLP | Recursive Length Prefix | 재귀 길이 접두사 |
| RPC | Remote Procedure Call | 원격 프로시저 호출 |
| SST | Sorted String Table | 정렬된 문자열 테이블 |
| TPS | Transactions Per Second | 초당 트랜잭션 |
| TPU | Transaction Processing Unit | 트랜잭션 처리 유닛 |
| TVU | Transaction Validation Unit | 트랜잭션 검증 유닛 |
| UID | Unique Identifier | 고유 식별자 |
| UTXO | Unspent Transaction Output | 미사용 트랜잭션 출력 |
| VDF | Verifiable Delay Function | 검증 가능 지연 함수 |
| VM | Virtual Machine | 가상 머신 |
| WAL | Write-Ahead Log | 선행 기록 로그 |
| ZK | Zero-Knowledge | 영지식 |

---

## 참고 자료

### 더 깊이 학습하기

- **Ethereum**: [ethereum.org/glossary](https://ethereum.org/en/glossary/)
- **Solana**: [docs.solana.com/terminology](https://docs.solana.com/terminology)
- **Sui**: [docs.sui.io/concepts](https://docs.sui.io/concepts)

### 논문

- **Merkle Trees**: "A Digital Signature Based on a Conventional Encryption Function" (Merkle, 1987)
- **Patricia Trie**: "Practical Algorithm to Retrieve Information Coded in Alphanumeric" (Morrison, 1968)
- **Kademlia**: "Kademlia: A Peer-to-peer Information System Based on the XOR Metric" (Maymounkov & Mazières, 2002)
- **PBFT**: "Practical Byzantine Fault Tolerance" (Castro & Liskov, 1999)

---

**이 용어 사전은 계속 업데이트됩니다!**

새로운 용어나 수정이 필요한 부분이 있다면 기여해주세요.
