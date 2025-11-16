# 코드 읽기 가이드

실제 블록체인 소스 코드를 효과적으로 읽고 이해하는 방법

---

## 🎯 목표

이 가이드는 다음을 도와줍니다:
- 수십만 줄의 코드에서 길을 잃지 않기
- 핵심 파일을 빠르게 찾기
- 데이터 흐름을 효과적으로 추적하기
- 새로운 코드베이스에 빠르게 적응하기

---

## 1. Ethereum (go-ethereum)

### 저장소 클론

```bash
git clone https://github.com/ethereum/go-ethereum.git
cd go-ethereum

# 특정 버전 체크아웃 (안정 버전)
git checkout v1.13.5

# 코드 통계
cloc .
```

### 디렉토리 구조 이해

```
go-ethereum/
├── p2p/              # 🔥 P2P 네트워킹 (1순위)
├── core/             # 🔥 블록체인 코어 (1순위)
│   ├── state/       # 🔥 상태 관리 (1순위)
│   ├── vm/          # EVM
│   ├── rawdb/       # 데이터베이스
│   └── types/       # 데이터 타입
├── trie/             # 🔥 Merkle Patricia Trie (1순위)
├── eth/              # Ethereum 프로토콜
├── ethdb/            # 데이터베이스 인터페이스
├── rpc/              # 🔥 JSON-RPC (1순위)
├── consensus/        # 합의 (PoS)
├── miner/            # 블록 생성
├── accounts/         # 계정 관리
├── crypto/           # 암호학
└── cmd/geth/         # Geth 실행 파일
```

### 핵심 파일 읽기 순서

#### Week 1: 기본 구조

**1일차: 블록체인 기본**
```bash
# 블록 구조
core/types/block.go
core/types/transaction.go
core/types/receipt.go

# 읽는 방법:
# 1. type 정의부터 (Block, Header, Transaction)
# 2. 주요 메서드 (Hash(), Size(), WithBody())
# 3. 직렬화/역직렬화 (RLP)
```

**2일차: 블록체인 관리**
```bash
core/blockchain.go

# 중점:
# - BlockChain 구조체
# - InsertChain() - 블록 삽입
# - GetBlockByHash(), GetBlockByNumber()
# - CurrentBlock(), CurrentHeader()

# 읽기 팁:
# 1. 구조체 필드부터
# 2. New* 생성자
# 3. 주요 public 메서드
# 4. 내부 helper 메서드는 나중에
```

**3일차: 상태 관리**
```bash
core/state/statedb.go

# 핵심:
# - StateDB 구조체
# - GetOrNewStateObject()
# - Commit()
# - Snapshot(), RevertToSnapshot()

# 주의:
# - Copy-on-Write 패턴
# - Journal for rollback
# - 캐싱 메커니즘
```

#### Week 2: 네트워킹

**4일차: P2P 기본**
```bash
p2p/server.go
p2p/peer.go

# 순서:
# 1. Server 구조체 이해
# 2. Start() 메서드
# 3. run() 메인 루프
# 4. Peer 관리 (addpeer, delpeer)
```

**5일차: RLPx 암호화**
```bash
p2p/rlpx/rlpx.go

# 중점:
# - Handshake() 과정
# - doEncHandshake(), doProtoHandshake()
# - Read(), Write() 암호화
```

**6일차: 노드 발견**
```bash
p2p/discover/udp.go
p2p/discover/table.go

# 핵심:
# - UDPv4 구조체
# - ping(), findnode()
# - Table (K-bucket)
```

#### Week 3: Trie

**7일차: Trie 기본**
```bash
trie/trie.go

# 읽는 순서:
# 1. node 타입들 (fullNode, shortNode, hashNode, valueNode)
# 2. Trie 구조체
# 3. Get(), Update()
# 4. Hash() - 해시 계산

# 디버깅:
# - 간단한 값 삽입하고 구조 출력
# - 해시 계산 과정 추적
```

**8일차: Secure Trie**
```bash
trie/secure_trie.go
trie/database.go

# 중점:
# - Keccak256 키 해싱
# - Preimage 관리
# - Database 캐싱
```

**9일차: Snapshot**
```bash
core/state/snapshot/snapshot.go
core/state/snapshot/difflayer.go

# 이해:
# - Tree 구조
# - diffLayer vs diskLayer
# - Account(), Storage() 조회
```

#### Week 4: RPC

**10일차: RPC 서버**
```bash
rpc/server.go
rpc/service.go

# 순서:
# 1. Server 구조체
# 2. RegisterName() - 서비스 등록
# 3. serveRequest() - 요청 처리
# 4. 리플렉션 사용 이해
```

**11일차: ETH API**
```bash
internal/ethapi/api.go

# 핵심 메서드:
# - BlockNumber()
# - GetBalance()
# - SendTransaction()
# - Call() - 시뮬레이션

# 각 메서드의 흐름:
# 1. 파라미터 검증
# 2. 상태/블록 조회
# 3. 연산 수행
# 4. 응답 형식 변환
```

**12일차: 필터 & 구독**
```bash
eth/filters/filter_system.go
eth/filters/api.go

# 이해:
# - EventSystem
# - Subscription 관리
# - 이벤트 전달 메커니즘
```

### 데이터 흐름 추적

**트랜잭션 -> 블록 전체 과정**

```
1. RPC 수신
   rpc/server.go: serveRequest()
   ↓
2. eth_sendTransaction 호출
   internal/ethapi/api.go: SendTransaction()
   ↓
3. TxPool에 추가
   core/tx_pool.go: AddLocal()
   ↓
4. Miner가 선택
   miner/worker.go: commitTransactions()
   ↓
5. EVM 실행
   core/state_processor.go: Process()
   core/vm/evm.go: Call()
   ↓
6. 상태 업데이트
   core/state/statedb.go: Finalise()
   ↓
7. 블록 생성
   miner/worker.go: commit()
   core/types/block.go: NewBlock()
   ↓
8. 블록 전파
   eth/handler.go: BroadcastBlock()
   ↓
9. 피어 수신
   eth/handler.go: handleMsg(NewBlockMsg)
   ↓
10. 블록 검증
   core/blockchain.go: InsertChain()
   ↓
11. 상태 커밋
   core/state/statedb.go: Commit()
   trie/trie.go: Commit()
   ↓
12. 데이터베이스 저장
   core/rawdb/accessors_chain.go: WriteBlock()
```

### 디버깅 팁

```go
// 로깅 추가
import "github.com/ethereum/go-ethereum/log"

log.Info("Current block", "number", block.Number(), "hash", block.Hash())
log.Debug("State object", "addr", addr, "balance", obj.Balance())

// 조건부 로깅
if log.Root().GetHandler() != nil {
    log.Trace("Detail info", "data", someData)
}

// 실행:
geth --verbosity 5  # 0=silent, 5=detail
```

**브레이크포인트 (Delve)**
```bash
# 설치
go install github.com/go-delve/delve/cmd/dlv@latest

# 디버깅
dlv exec ./build/bin/geth -- --datadir ./mydata

# 브레이크포인트
(dlv) break core/state/statedb.go:100
(dlv) continue
(dlv) print obj
(dlv) next
```

---

## 2. Solana (Agave)

### 저장소 클론

```bash
git clone https://github.com/anza-xyz/agave.git
cd agave

# 안정 버전
git checkout v1.18.0

# 의존성 설치
cargo build --release
```

### 디렉토리 구조

```
agave/
├── poh/              # 🔥 Proof of History (1순위)
├── gossip/           # 🔥 Gossip 프로토콜 (1순위)
├── ledger/           # 🔥 Blockstore (1순위)
│   └── src/
│       ├── blockstore.rs
│       └── shred.rs
├── runtime/          # 🔥 실행 런타임 (1순위)
│   └── src/
│       ├── bank.rs
│       └── accounts_db.rs
├── core/             # 🔥 코어 파이프라인 (1순위)
│   └── src/
│       ├── banking_stage.rs
│       ├── broadcast_stage.rs
│       └── retransmit_stage.rs
├── rpc/              # JSON-RPC
├── programs/         # 네이티브 프로그램
└── sdk/              # SDK
```

### 핵심 파일 읽기 순서

#### Week 1: PoH

**1일차: PoH 기본**
```bash
poh/src/poh_recorder.rs
poh/src/poh_service.rs

# 중점:
# - Poh 구조체
# - hash() - 단일 해시
# - tick() - 틱 생성
# - record() - 트랜잭션 기록

# 실습:
# examples/solana/simple-poh.rs 함께 보기
```

**2일차: Entry**
```bash
entry/src/entry.rs

# 이해:
# - Entry 구조체
# - verify() - 검증
# - create_ticks() - 틱 생성
```

#### Week 2: Gossip

**3일차: ClusterInfo**
```bash
gossip/src/cluster_info.rs

# 순서:
# 1. ClusterInfo 구조체
# 2. CrdsValue 타입들
# 3. gossip_loop()
# 4. push/pull 메커니즘
```

**4일차: CRDS**
```bash
gossip/src/crds.rs
gossip/src/crds_gossip_push.rs
gossip/src/crds_gossip_pull.rs

# 중점:
# - CRDS 구조체
# - insert() - 데이터 삽입
# - get() - 데이터 조회
# - 벡터 클락 이해
```

#### Week 3: Shred & Turbine

**5일차: Shred**
```bash
ledger/src/shred.rs

# 핵심:
# - Shred 구조체
# - ShredCommonHeader, DataShredHeader, CodingShredHeader
# - new_from_data() - 생성
# - payload() - 데이터

# 주의:
# - Data vs Coding shred 차이
# - FEC 세트 개념
```

**6일차: Shredder**
```bash
ledger/src/shredder.rs

# 순서:
# 1. Shredder 구조체
# 2. entries_to_shreds() - Entry를 Shred로
# 3. generate_coding_shreds() - FEC 생성
```

**7일차: BroadcastStage**
```bash
core/src/broadcast_stage/broadcast_stage.rs
core/src/broadcast_stage/standard_broadcast_run.rs

# 이해:
# - Turbine tree 구성
# - broadcast_shreds()
# - DATA_PLANE_FANOUT
```

**8일차: RetransmitStage**
```bash
core/src/retransmit_stage.rs

# 중점:
# - 재전파 로직
# - 자식 노드 계산
# - Shred 복구 (FEC)
```

#### Week 4: AccountsDB

**9일차: AccountsDB 기본**
```bash
runtime/src/accounts_db.rs

# 방대한 파일! 부분별로:
# 1. AccountsDb 구조체 (줄 ~500)
# 2. load() 메서드
# 3. store() 메서드
# 4. flush() - 디스크 쓰기
```

**10일차: AppendVec**
```bash
runtime/src/append_vec.rs

# 순서:
# 1. AppendVec 구조체
# 2. append_account()
# 3. get_account()
# 4. 메모리 맵 이해
```

**11일차: AccountsIndex**
```bash
runtime/src/accounts_index.rs

# 핵심:
# - AccountsIndex 구조체
# - upsert()
# - get()
# - 슬롯 리스트 관리
```

**12일차: Banking Stage**
```bash
core/src/banking_stage.rs

# 복잡! 단계별:
# 1. BankingStage 구조체
# 2. process_packets() - 메인 로직
# 3. lock_accounts() - 충돌 감지
# 4. create_non_conflicting_batches()
# 5. execute_batch() - 병렬 실행
```

### 데이터 흐름 추적

**트랜잭션 -> 슬롯 전체 과정**

```
1. RPC 수신
   rpc/src/rpc.rs: send_transaction()
   ↓
2. TPU로 UDP 전송
   (Transaction Processing Unit)
   ↓
3. FetchStage: 수신
   core/src/fetch_stage.rs
   ↓
4. SigVerifyStage: 서명 검증 (GPU)
   core/src/sigverify_stage.rs
   ↓
5. BankingStage: 실행
   core/src/banking_stage.rs
   - 계정 충돌 감지
   - 병렬 배치 실행
   ↓
6. Bank: 상태 업데이트
   runtime/src/bank.rs: commit_transactions()
   ↓
7. AccountsDB: 저장
   runtime/src/accounts_db.rs: store()
   ↓
8. PohRecorder: 기록
   poh/src/poh_recorder.rs: record()
   ↓
9. Entry 생성
   entry/src/entry.rs
   ↓
10. BroadcastStage: Shred 생성
   core/src/broadcast_stage/: entries_to_shreds()
   ↓
11. Turbine: 전파
   스테이크 순으로 fanout
   ↓
12. RetransmitStage: 재전파
   core/src/retransmit_stage.rs
   ↓
13. Blockstore: 저장
   ledger/src/blockstore.rs: insert_shreds()
```

### 디버깅 팁

```rust
// 로깅
use log::{info, debug, trace};

info!("Processing slot {}", slot);
debug!("Shred count: {}", shreds.len());
trace!("Account data: {:?}", account);

// 실행
RUST_LOG=solana=debug cargo run

// 조건부 로깅
RUST_LOG=solana_core::banking_stage=trace cargo run

// 테스트
cargo test test_name -- --nocapture

// 벤치마크
cargo bench
```

**Rust Analyzer (VSCode)**
```json
{
  "rust-analyzer.cargo.features": "all",
  "rust-analyzer.checkOnSave.command": "clippy"
}
```

---

## 3. Sui

### 저장소 클론

```bash
git clone https://github.com/MystenLabs/sui.git
cd sui

# 안정 버전
git checkout mainnet

# 빌드
cargo build --release
```

### 디렉토리 구조

```
sui/
├── narwhal/          # 🔥 합의 엔진 (1순위)
│   ├── primary/
│   ├── worker/
│   └── types/
├── crates/
│   ├── sui-types/   # 🔥 타입 정의 (1순위)
│   ├── sui-core/    # 🔥 코어 로직 (1순위)
│   ├── sui-json-rpc/ # RPC
│   ├── sui-network/  # 네트워킹
│   └── sui-framework/ # 🔥 Move 프레임워크 (1순위)
└── external-crates/
    └── move/         # Move VM
```

### 핵심 파일 읽기 순서

#### Week 1: 객체 모델

**1일차: Object**
```bash
crates/sui-types/src/object.rs

# 순서:
# 1. Object 구조체
# 2. Owner enum
# 3. Data enum (Move, Package)
# 4. 주요 메서드들
```

**2일차: Transaction**
```bash
crates/sui-types/src/transaction.rs
crates/sui-types/src/programmable_transaction_builder.rs

# 중점:
# - TransactionData
# - TransactionKind
# - ProgrammableTransaction
# - Command enum
```

**3일차: Effects**
```bash
crates/sui-types/src/effects.rs

# 이해:
# - TransactionEffects
# - created, mutated, deleted
# - 이벤트
```

#### Week 2: Storage

**4일차: AuthorityStore**
```bash
crates/sui-core/src/authority/authority_store.rs

# 방대! 부분별로:
# 1. AuthorityStore 구조체
# 2. get_object()
# 3. update_objects()
# 4. 인덱스들
```

**5일차: TypedStore**
```bash
crates/typed-store/src/rocks/mod.rs

# 핵심:
# - DBMap<K, V>
# - insert(), get()
# - multi_insert()
# - iter()
```

#### Week 3: Narwhal

**6일차: Header & Certificate**
```bash
narwhal/types/src/header.rs
narwhal/types/src/certificate.rs

# 순서:
# 1. Header 구조체
# 2. Certificate 구조체
# 3. verify() 메서드들
```

**7일차: Primary**
```bash
narwhal/primary/src/primary.rs
narwhal/primary/src/core.rs

# 중점:
# - Primary 구조체
# - propose_header()
# - process_header()
# - process_vote()
```

**8일차: Worker**
```bash
narwhal/worker/src/worker.rs

# 이해:
# - Worker 구조체
# - make_batch()
# - 배치 스토리지
```

**9일차: Tusk**
```bash
narwhal/consensus/src/tusk.rs

# 핵심:
# - commit() - 순서 결정
# - select_anchor()
# - DAG 순회
```

#### Week 4: Move & Execution

**10일차: Move VM**
```bash
external-crates/move/move-vm/runtime/src/lib.rs

# 기본만:
# - MoveVM 구조체
# - execute_function()
# - 세션 관리
```

**11일차: Sui Framework**
```bash
crates/sui-framework/packages/sui-framework/sources/

# 주요 모듈:
# - coin.move
# - transfer.move
# - object.move
# - tx_context.move

# 읽는 법:
# 1. struct 정의
# 2. public entry 함수들
# 3. public 함수들
# 4. 내부 함수들
```

**12일차: Transaction Execution**
```bash
crates/sui-core/src/execution_engine.rs
crates/sui-core/src/transaction_manager.rs

# 흐름:
# 1. execute_transaction()
# 2. FastPath vs ConsensusPath 판단
# 3. Move 함수 호출
# 4. Effects 생성
```

### 데이터 흐름 추적

**트랜잭션 -> 체크포인트 (FastPath)**

```
1. RPC 수신
   crates/sui-json-rpc/src/api.rs: execute_transaction_block()
   ↓
2. TransactionOrchestrator
   트랜잭션 검증 및 라우팅
   ↓
3. FastPath 판단
   (모든 입력이 소유 객체?)
   ↓
4. AuthorityState: 즉시 실행
   crates/sui-core/src/authority/authority.rs
   - 소유권 검증
   - Move VM 실행
   ↓
5. Move VM
   external-crates/move/move-vm/: execute_function()
   ↓
6. 상태 업데이트
   crates/sui-core/src/authority/authority_store.rs: update_objects()
   ↓
7. Effects 생성
   TransactionEffects
   ↓
8. Certificate 발급
   (단일 검증자 서명, 합의 불필요!)
   ↓
9. 클라이언트에 반환
   ↓
(백그라운드)
10. CheckpointBuilder
   트랜잭션들 묶기
   ↓
11. 체크포인트 생성
   검증자 서명 수집
   ↓
12. 저장
   authority_store.rs: insert_checkpoint()
```

### 디버깅 팁

```rust
// 로깅
use tracing::{info, debug, trace};

info!("Processing object {}", object_id);
debug!("Owner: {:?}", owner);
trace!("Full object: {:?}", object);

// 실행
RUST_LOG=sui=debug cargo run

// Move 디버깅
sui move test --coverage

// 프로파일링
cargo flamegraph
```

---

## 공통 팁

### IDE 설정

**VSCode**:
```json
{
  "editor.formatOnSave": true,
  "editor.rulers": [100],

  // Go
  "go.lintTool": "golangci-lint",
  "go.formatTool": "goimports",

  // Rust
  "rust-analyzer.checkOnSave.command": "clippy",
  "rust-analyzer.cargo.features": "all"
}
```

### 코드 탐색 단축키

| 기능 | VSCode | IntelliJ |
|------|--------|----------|
| 정의로 이동 | F12 | Cmd+B |
| 참조 찾기 | Shift+F12 | Cmd+Alt+F7 |
| 심볼 검색 | Cmd+T | Cmd+Alt+O |
| 파일 검색 | Cmd+P | Cmd+Shift+O |
| 호출 계층 | Shift+Alt+H | Ctrl+Alt+H |

### 주석 달기 템플릿

```
파일 상단:
// <파일명>
// 목적: <주요 역할>
// 의존성: <주요 의존 모듈>
// 핵심 개념: <이해해야 할 개념>

구조체/타입:
// <이름>
// 역할: <무엇을 하는가>
// 패턴: <디자인 패턴>
// 주의: <함정, 제약사항>
// 예시:
//   <간단한 사용 예>

함수:
// <이름>
// 목적: <왜 필요한가>
// 흐름:
//   1. <첫 번째 단계>
//   2. <두 번째 단계>
//   ...
// 복잡도: O(?)
// 부작용: <상태 변경 등>
```

### 학습 프로젝트 아이디어

**작은 프로젝트로 시작**:
1. 특정 데이터 구조 추출 (Merkle tree, Shred 등)
2. 독립 실행 가능하게 만들기
3. 테스트 작성
4. 벤치마크

**예시**:
```bash
# Ethereum Trie 추출
mkdir my-trie
cp -r go-ethereum/trie my-trie/
# 의존성 최소화
# 테스트 실행
# 성능 측정
```

---

## 마무리

### 학습 체크리스트

- [ ] 각 체인의 디렉토리 구조 이해
- [ ] 핵심 파일 10개 이상 읽기
- [ ] 데이터 흐름 추적 가능
- [ ] 디버거 사용 가능
- [ ] 간단한 기능 추가 가능

### 다음 단계

1. **이슈 트래킹**: GitHub issues 읽기
2. **PR 리뷰**: 다른 사람의 코드 리뷰
3. **기여**: 문서 개선, 버그 수정
4. **블로그**: 학습 내용 정리

**행운을 빕니다! 🚀**
