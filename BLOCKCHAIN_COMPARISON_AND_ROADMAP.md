# 이더리움, 솔라나, Sui 완전 비교 및 학습 로드맵

## 목차
1. [아키텍처 비교](#1-아키텍처-비교)
2. [실제 코드 구조 비교](#2-실제-코드-구조-비교)
3. [성능 특성 비교](#3-성능-특성-비교)
4. [학습 우선순위 및 로드맵](#4-학습-우선순위-및-로드맵)
5. [실습 프로젝트](#5-실습-프로젝트)

---

## 1. 아키텍처 비교

### 1.1 네트워크 레이어

| 구분 | Ethereum | Solana | Sui |
|------|----------|--------|-----|
| **프로토콜** | DevP2P (RLPx) | Gossip + Turbine | Anemo (Noise) + Narwhal |
| **전송 방식** | TCP (암호화) | UDP | TCP (암호화) |
| **노드 발견** | Kademlia DHT (UDP) | Gossip PlumTree | Anemo discovery |
| **데이터 전파** | Full block flooding | Shred + Turbine tree | DAG-based (Narwhal) |
| **구조** | Flat P2P | Multi-layer tree | Primary-Worker |

#### 상세 비교

**Ethereum (DevP2P)**
```
장점:
- 성숙한 프로토콜 (10년+ 운영)
- 강력한 암호화 (ECIES + AES-256)
- 노드 발견 및 연결 안정적

단점:
- 블록 전파가 느림 (15초 블록시간의 주요 원인)
- 대역폭 비효율적 (모든 노드가 전체 블록 수신)

핵심 코드:
- p2p/server.go: 피어 연결 관리
- p2p/rlpx/rlpx.go: 암호화 핸드셰이크
- p2p/discover/: Kademlia DHT
```

**Solana (Turbine)**
```
장점:
- 매우 빠른 전파 (200ms 이내)
- Reed-Solomon FEC로 손실 복구
- 스테이크 가중 전파 순서

단점:
- UDP 기반이라 신뢰성 낮음
- 네트워크 혼잡시 패킷 손실

핵심 코드:
- ledger/src/shred.rs: Shred 생성
- core/src/broadcast_stage/: Turbine tree 전파
- gossip/src/cluster_info.rs: 제어 평면
```

**Sui (Narwhal)**
```
장점:
- 전파와 합의 분리 (높은 처리량)
- DAG 구조로 병렬 전파
- Primary-Worker로 부하 분산

단점:
- 복잡한 아키텍처
- 메모리 사용량 높음

핵심 코드:
- narwhal/primary/src/: Primary 노드
- narwhal/worker/src/: Worker 노드
- narwhal/types/src/: Header, Certificate
```

### 1.2 합의 알고리즘

| 구분 | Ethereum | Solana | Sui |
|------|----------|--------|-----|
| **알고리즘** | Gasper (PoS) | Tower BFT + PoH | Narwhal-Tusk / Mysticeti |
| **Finality** | 2 epochs (~13분) | 32 slots (~13초) | 3-5초 (체크포인트) |
| **리더** | Proposer rotation | PoH 기반 스케줄 | 없음 (DAG) / Mysticeti |
| **처리량** | 15-30 TPS | 50,000+ TPS | 100,000+ TPS |
| **검증자 수** | 무제한 | 3,000+ | 100-150 (현재) |

#### 상세 비교

**Ethereum (Gasper)**
```
특징:
- Casper FFG (Finality) + LMD GHOST (포크 선택)
- 슬롯 12초, 에포크 32 슬롯
- 검증자 위원회 로테이션

장점:
- 매우 안전 (수학적 증명)
- 탈중앙화 (수십만 검증자)

단점:
- 느린 finality
- 낮은 처리량

핵심 개념:
- Attestation: 검증자 투표
- Justification: 2/3 투표 받은 체크포인트
- Finalization: 연속 2개 justified
```

**Solana (Tower BFT)**
```
특징:
- PoH로 시간 경과 증명
- 리더 스케줄 미리 결정
- Lockout으로 투표 가중치

장점:
- 빠른 finality
- 높은 처리량

단점:
- 중단 위험 (과거 여러 차례 중단)
- 높은 하드웨어 요구사항

핵심 개념:
- PoH: SHA-256 해시 체인
- Tower: 투표 타워 (스택)
- Lockout: 2^confirmation_count
```

**Sui (Narwhal-Tusk → Mysticeti)**
```
특징:
- DAG 기반 mempool (Narwhal)
- Zero-message overhead 합의 (Tusk)
- 소유 객체는 합의 불필요 (FastPath)

장점:
- 매우 높은 처리량
- 간단한 트랜잭션 즉시 처리

단점:
- 복잡한 구조
- 아직 검증 중

핵심 개념:
- Header: DAG 정점
- Certificate: 2/3+ 서명
- Anchor: 순서 결정 기준
```

### 1.3 데이터베이스

| 구분 | Ethereum | Solana | Sui |
|------|----------|--------|-----|
| **메인 DB** | LevelDB | RocksDB | RocksDB |
| **키-값 스토어** | 예 | 예 | 예 (TypedStore) |
| **상태 모델** | Account (MPT) | Account (AppendVec) | Object |
| **히스토리** | Freezer (ancient) | Ledger | Object versions |
| **인덱싱** | 블록/트랜잭션 | 트랜잭션/계정 | 객체/소유자 |

#### 상세 비교

**Ethereum (LevelDB + Freezer)**
```
구조:
1. LevelDB: 최근 블록 (빠른 접근)
2. Freezer: 오래된 블록 (압축 저장)

키 스킴:
- "h" + num + hash -> header
- "b" + num + hash -> body
- "r" + num + hash -> receipts
- "H" + hash -> block number

장점:
- 안정적이고 검증됨
- 공간 효율적 (Freezer)

단점:
- 동기화 느림 (수일 소요)
- 디스크 사용량 많음 (수 TB)

핵심 코드:
- core/rawdb/: 키 스킴, 접근 함수
- core/rawdb/freezer.go: Ancient 저장소
```

**Solana (RocksDB)**
```
구조:
1. Blockstore: Shred 저장 (Column Families)
2. AccountsDB: 계정 상태 (AppendVec)

Column Families:
- ShredData / ShredCode
- SlotMeta
- TransactionStatus
- Rewards

장점:
- 매우 빠른 쓰기 (Append-only)
- 스냅샷 지원

단점:
- 디스크 사용량 폭발 (400TB+)
- 압축 필요

핵심 코드:
- ledger/src/blockstore_db.rs: RocksDB 래퍼
- runtime/src/accounts_db.rs: AppendVec
```

**Sui (RocksDB + TypedStore)**
```
구조:
1. TypedStore: 타입 안전 래퍼
2. AuthorityStore: 객체/트랜잭션 저장

DBMap들:
- objects: ObjectID -> Object
- transactions: Digest -> Transaction
- effects: Digest -> Effects
- owner_index: Owner -> Vec<ObjectID>

장점:
- 타입 안전성
- 객체 버전 관리
- 소유권 인덱스

단점:
- 아직 성숙도 낮음
- Pruning 필수

핵심 코드:
- crates/typed-store/: TypedStore
- crates/sui-core/src/authority/authority_store.rs
```

### 1.4 상태 관리

| 구분 | Ethereum | Solana | Sui |
|------|----------|--------|-----|
| **모델** | Account-based | Account-based | Object-based |
| **자료구조** | Merkle Patricia Trie | Flat (AppendVec) | Object Graph |
| **상태 루트** | Keccak256 hash | Accounts Hash | 체크포인트 |
| **검증** | Merkle proof | Merkle proof | Object digest |
| **병렬성** | 없음 (순차) | Sealevel (계정 충돌) | 객체 소유권 |

#### 상세 비교

**Ethereum (Merkle Patricia Trie)**
```
구조:
- World State Trie: 주소 -> 계정
- Storage Trie: 각 계정의 스토리지
- Transaction Trie: 블록의 트랜잭션들
- Receipt Trie: 트랜잭션 영수증

노드 타입:
- fullNode: 16개 자식 (hex 문자)
- shortNode: 경로 압축
- hashNode: 해시 참조
- valueNode: 실제 값

장점:
- 암호학적으로 안전
- Merkle proof 지원
- 경로 압축으로 공간 절약

단점:
- 느린 접근 (디스크 I/O 많음)
- 순차 실행만 가능

핵심 코드:
- trie/trie.go: Trie 구조
- trie/secure_trie.go: Keccak256 키
- core/state/statedb.go: 상태 관리
```

**Solana (AccountsDB)**
```
구조:
- AppendVec: Append-only 계정 파일
- AccountsIndex: Pubkey -> (슬롯, 위치)
- AccountsCache: 최근 업데이트 캐시

특징:
- 메모리 맵 파일 (mmap)
- Copy-on-Write
- 백그라운드 압축

병렬 실행:
- 트랜잭션이 접근하는 계정 명시
- 충돌 없는 트랜잭션 병렬 처리
- Banking Stage에서 배치 실행

장점:
- 매우 빠른 읽기/쓰기
- 병렬 실행 가능

단점:
- 메모리 사용량 높음
- 압축 필요

핵심 코드:
- runtime/src/accounts_db.rs: AccountsDB
- runtime/src/append_vec.rs: AppendVec
- core/src/banking_stage.rs: 병렬 실행
```

**Sui (Object Model)**
```
구조:
- Object: ID, Owner, Version, Data
- Owner 타입:
  - AddressOwner: 주소 소유
  - ObjectOwner: 객체 소유
  - Shared: 공유
  - Immutable: 불변

병렬 실행:
- 소유 객체: 합의 불필요 (FastPath)
- 공유 객체: 합의 필요 (ConsensusPath)

장점:
- 직관적인 모델
- 최고의 병렬성
- 소유권 명확

단점:
- 새로운 패러다임 (학습 곡선)
- Move 언어 필요

핵심 코드:
- crates/sui-types/src/object.rs: Object 정의
- crates/sui-core/src/authority/authority_store.rs: 저장
```

### 1.5 RPC & API

| 구분 | Ethereum | Solana | Sui |
|------|----------|--------|-----|
| **프로토콜** | JSON-RPC 2.0 | JSON-RPC 2.0 | JSON-RPC 2.0 |
| **전송** | HTTP, WS, IPC | HTTP, WS | HTTP, WS |
| **네임스페이스** | eth, net, web3, debug | getAccountInfo, sendTransaction | sui_*, suix_* |
| **구독** | eth_subscribe (WS) | accountSubscribe (WS) | subscribeEvent (WS) |
| **특징** | 필터, 로그 | 계정 변경, 슬롯 | 이벤트, 객체 |

---

## 2. 실제 코드 구조 비교

### 2.1 저장소 구조

**Ethereum (go-ethereum)**
```
go-ethereum/
├── p2p/                    # P2P 네트워킹
│   ├── server.go          # 피어 관리
│   ├── peer.go            # 피어 연결
│   ├── rlpx/              # 암호화 전송
│   └── discover/          # 노드 발견
├── core/                   # 블록체인 코어
│   ├── blockchain.go      # 블록체인 구조
│   ├── state/             # 상태 관리
│   │   ├── statedb.go    # StateDB
│   │   └── snapshot/     # 스냅샷
│   ├── rawdb/             # 데이터베이스
│   │   ├── schema.go     # 키 스킴
│   │   └── freezer.go    # Ancient 저장
│   └── vm/                # EVM
├── eth/                    # Ethereum 프로토콜
│   ├── handler.go         # 프로토콜 핸들러
│   └── protocols/eth/     # ETH 프로토콜
├── trie/                   # Merkle Patricia Trie
│   ├── trie.go
│   └── secure_trie.go
├── ethdb/                  # 데이터베이스 인터페이스
├── rpc/                    # JSON-RPC
│   ├── server.go
│   └── types.go
└── consensus/              # 합의
    ├── beacon/            # PoS (Beacon chain)
    └── ethash/            # PoW (legacy)

핵심 학습 파일 (우선순위):
1. p2p/server.go - 네트워크 기초
2. core/blockchain.go - 블록체인 구조
3. trie/trie.go - 상태 저장 이해
4. core/state/statedb.go - 상태 관리
5. rpc/server.go - RPC 이해
```

**Solana (agave)**
```
agave/
├── gossip/                 # Gossip 프로토콜
│   └── src/
│       ├── cluster_info.rs # CRDS
│       └── gossip_service.rs
├── poh/                    # Proof of History
│   └── src/
│       └── poh_recorder.rs
├── ledger/                 # Ledger & Blockstore
│   └── src/
│       ├── blockstore_db.rs # RocksDB
│       └── shred.rs        # Shred 정의
├── runtime/                # 런타임
│   └── src/
│       ├── accounts_db.rs  # AccountsDB
│       ├── append_vec.rs   # AppendVec
│       └── bank.rs         # Bank (상태)
├── core/                   # 코어 파이프라인
│   └── src/
│       ├── banking_stage.rs # 트랜잭션 실행
│       ├── broadcast_stage.rs # Turbine
│       ├── retransmit_stage.rs
│       ├── tvu.rs          # Transaction Validation Unit
│       └── tpu.rs          # Transaction Processing Unit
├── rpc/                    # JSON-RPC
│   └── src/
│       ├── rpc.rs
│       └── rpc_pubsub.rs
└── programs/               # 네이티브 프로그램
    ├── bpf_loader/
    └── system/

핵심 학습 파일 (우선순위):
1. poh/src/poh_recorder.rs - PoH 이해
2. ledger/src/shred.rs - 데이터 구조
3. core/src/banking_stage.rs - 실행 파이프라인
4. runtime/src/accounts_db.rs - 상태 관리
5. gossip/src/cluster_info.rs - 네트워크
```

**Sui**
```
sui/
├── narwhal/                # Narwhal 합의 (별도)
│   ├── primary/
│   │   └── src/
│   │       └── primary.rs
│   ├── worker/
│   │   └── src/
│   │       └── worker.rs
│   └── types/
│       └── src/
│           ├── header.rs
│           └── certificate.rs
├── crates/
│   ├── sui-core/          # 코어 로직
│   │   └── src/
│   │       ├── authority/
│   │       │   └── authority_store.rs # 메인 저장소
│   │       └── checkpoints/
│   ├── sui-types/         # 타입 정의
│   │   └── src/
│   │       ├── object.rs  # Object 모델
│   │       ├── transaction.rs
│   │       └── effects.rs
│   ├── sui-json-rpc/      # JSON-RPC
│   │   └── src/
│   │       ├── api.rs
│   │       └── read_api.rs
│   ├── sui-network/       # 네트워킹
│   │   └── src/
│   ├── sui-framework/     # Move 프레임워크
│   │   └── packages/
│   │       └── sui-framework/
│   └── typed-store/       # 데이터베이스
│       └── src/
│           └── rocks/
└── external-crates/
    └── move/              # Move VM
        └── move-vm/

핵심 학습 파일 (우선순위):
1. crates/sui-types/src/object.rs - 객체 모델
2. crates/sui-core/src/authority/authority_store.rs - 저장소
3. narwhal/types/src/header.rs - 합의 구조
4. crates/sui-json-rpc/src/api.rs - RPC
5. crates/sui-framework/ - Move 프레임워크
```

### 2.2 핵심 데이터 흐름

**Ethereum: 트랜잭션 -> 블록**
```
1. 사용자 -> RPC (eth_sendTransaction)
   rpc/server.go: serveRequest()

2. RPC -> TxPool
   core/tx_pool.go: AddLocal()

3. TxPool -> Miner (Proposer)
   miner/worker.go: commitTransactions()

4. EVM 실행
   core/vm/evm.go: Call()

5. 상태 업데이트
   core/state/statedb.go: Commit()

6. 블록 생성
   core/types/block.go: NewBlock()

7. 블록 전파
   eth/handler.go: BroadcastBlock()

8. 피어 수신
   eth/handler.go: handleMsg(NewBlockMsg)

9. 블록 검증 및 삽입
   core/blockchain.go: InsertChain()

10. 상태 커밋
    core/state/statedb.go: Commit()
    trie/trie.go: Commit()
```

**Solana: 트랜잭션 -> 슬롯**
```
1. 사용자 -> RPC (sendTransaction)
   rpc/src/rpc.rs: send_transaction()

2. RPC -> TPU (리더)
   UDP 소켓으로 전송

3. FetchStage: 트랜잭션 수신
   core/src/fetch_stage.rs

4. SigVerifyStage: 서명 검증 (GPU)
   core/src/sigverify_stage.rs

5. BankingStage: 트랜잭션 실행
   core/src/banking_stage.rs
   - 계정 충돌 감지
   - 병렬 배치 실행

6. Bank: 상태 업데이트
   runtime/src/bank.rs: commit_transactions()

7. AccountsDB: 계정 저장
   runtime/src/accounts_db.rs: store()

8. PohRecorder: PoH에 기록
   poh/src/poh_recorder.rs: record()

9. BroadcastStage: Shred 생성 및 전파
   core/src/broadcast_stage/:
   - entries_to_shreds()
   - Turbine tree 전파

10. 피어 수신 (RetransmitStage)
    core/src/retransmit_stage.rs
    - Shred 재전파
    - FEC 복구

11. Blockstore: Shred 저장
    ledger/src/blockstore.rs: insert_shreds()
```

**Sui: 트랜잭션 -> 체크포인트**
```
1. 사용자 -> RPC (executeTransactionBlock)
   crates/sui-json-rpc/src/api.rs

2. RPC -> TransactionOrchestrator
   트랜잭션 검증 및 라우팅

3. FastPath vs ConsensusPath 판단
   - 소유 객체만? -> FastPath
   - 공유 객체 있음? -> ConsensusPath

=== FastPath (소유 객체만) ===
4a. AuthorityState: 즉시 실행
    crates/sui-core/src/authority/authority.rs
    - 소유권 검증
    - Move VM 실행

5a. 상태 업데이트
    authority_store.rs: update_objects()

6a. Certificate 발급 및 반환
    (합의 불필요!)

=== ConsensusPath (공유 객체) ===
4b. Narwhal: Worker에 배치 전송
    narwhal/worker/src/worker.rs

5b. Primary: Header 생성 및 전파
    narwhal/primary/src/primary.rs

6b. Vote 수집 -> Certificate 생성
    narwhal/primary/src/certificate_maker.rs

7b. Tusk: DAG 순서화
    narwhal/consensus/src/tusk.rs

8b. AuthorityState: 순서대로 실행
    트랜잭션 실행 및 Effects 생성

=== 공통 ===
9. CheckpointBuilder: 체크포인트 생성
   - 트랜잭션들 묶기
   - 검증자 서명 수집

10. 체크포인트 저장
    authority_store.rs: insert_checkpoint()
```

---

## 3. 성능 특성 비교

### 3.1 처리량 (TPS)

| 블록체인 | 이론 TPS | 실제 TPS | 제한 요인 |
|---------|---------|---------|----------|
| **Ethereum** | ~30 | 15-20 | EVM 순차 실행, 블록 가스 한도 |
| **Solana** | 65,000+ | 3,000-5,000 | 네트워크 대역폭, 상태 성장 |
| **Sui** | 120,000+ | 5,000-10,000 | 공유 객체 합의, 검증자 수 |

### 3.2 레이턴시 (Finality)

| 블록체인 | 블록 시간 | Finality | 실제 체감 |
|---------|---------|----------|----------|
| **Ethereum** | 12초 | ~13분 (2 epoch) | 1분 (확률적) |
| **Solana** | 400ms | ~13초 (32 slots) | 2-3초 (실용적) |
| **Sui** | N/A | 3-5초 (체크포인트) | 1초 (FastPath) |

### 3.3 하드웨어 요구사항

**Ethereum (Full Node)**
```
CPU: 4+ 코어
RAM: 16GB+ (권장 32GB)
디스크: 1TB+ SSD (계속 증가)
네트워크: 25 Mbps
동기화 시간: 수일 (snap sync: 수시간)

검증자 (Validator):
CPU: 4+ 코어
RAM: 32GB+
디스크: 2TB+ NVMe SSD
네트워크: 100 Mbps
추가: Beacon node + Execution client
```

**Solana (Validator)**
```
CPU: 12+ 코어 (AMD Zen3+)
RAM: 256GB+ (128GB minimum)
디스크: 2TB+ NVMe SSD (PCIe Gen4, 계속 증가)
네트워크: 1 Gbps
GPU: 권장 (서명 검증)
동기화 시간: 수일

비용: $5,000-10,000+

주의: 매우 높은 요구사항!
```

**Sui (Validator)**
```
CPU: 24+ 코어
RAM: 128GB+
디스크: 4TB+ NVMe SSD
네트워크: 1 Gbps
동기화 시간: 수시간 (스냅샷)

비용: $3,000-8,000

참고: 검증자 수 제한적 (허가형에 가까움)
```

### 3.4 개발자 경험

**Ethereum**
```
언어: Solidity, Vyper
툴: Hardhat, Foundry, Remix
학습 곡선: 중간
생태계: 최대 (가장 많은 자료)
디버깅: Tenderly, 풍부한 도구

장점:
- 가장 많은 학습 자료
- 성숙한 도구
- 표준화된 패턴 (ERC-20, ERC-721)

단점:
- Gas 최적화 어려움
- 재진입 공격 등 보안 이슈
```

**Solana**
```
언어: Rust (프로그램)
툴: Anchor, Solana CLI
학습 곡선: 높음 (Rust + Solana 개념)
생태계: 중간 (성장 중)
디버깅: 어려움

장점:
- 높은 성능
- 낮은 수수료

단점:
- Rust 필수
- 계정 모델 복잡
- 제한적인 도구
- 프로그램 업그레이드 제한
```

**Sui**
```
언어: Move
툴: Sui CLI, Sui Explorer
학습 곡선: 높음 (새로운 언어)
생태계: 작음 (초기 단계)
디버깅: 제한적

장점:
- 안전한 언어 (Move)
- 직관적인 객체 모델
- 병렬 실행 자동

단점:
- 새로운 언어 (학습 필요)
- 제한적인 자료
- 초기 단계
```

---

## 4. 학습 우선순위 및 로드맵

### 4.1 전체 학습 로드맵 (12주)

#### Week 1-2: 기초 개념
```
목표: 블록체인 기본 이해

공통 주제:
□ 블록체인 기본 (블록, 트랜잭션, 해시)
□ P2P 네트워킹 개념
□ 합의 알고리즘 개요
□ 데이터베이스 기초 (키-값 스토어)

실습:
□ 간단한 블록체인 구현 (Python/Go)
□ 해시 체인 만들기
□ Merkle Tree 구현

자료:
- Bitcoin whitepaper
- Ethereum whitepaper
- Mastering Bitcoin (책)
```

#### Week 3-4: Ethereum 심화
```
목표: Ethereum 내부 구조 이해

학습 순서:
1. DevP2P 네트워킹
   □ p2p/server.go 읽기
   □ RLPx 핸드셰이크 이해
   □ 로컬 2개 노드 연결 테스트

2. Merkle Patricia Trie
   □ trie/trie.go 분석
   □ 간단한 trie 직접 구현
   □ 해시 계산 과정 추적

3. StateDB
   □ core/state/statedb.go 읽기
   □ 트랜잭션 실행 과정 추적
   □ 상태 변경 시각화

4. RPC
   □ rpc/server.go 구조 분석
   □ 커스텀 RPC 메서드 추가
   □ Web3.js로 호출

실습 프로젝트:
□ 간단한 블록 익스플로러
□ 상태 변경 추적 도구
□ Merkle proof 검증기

코드 읽기 목표:
- 매일 최소 1개 파일 완전 이해
- 주석 달며 읽기
- 핵심 함수 호출 그래프 그리기
```

#### Week 5-6: Solana 심화
```
목표: Solana 고성능 아키텍처 이해

학습 순서:
1. Proof of History
   □ poh/src/poh_recorder.rs 분석
   □ PoH 생성 및 검증 실습
   □ 시간 경과 증명 이해

2. Turbine & Shreds
   □ ledger/src/shred.rs 구조
   □ Reed-Solomon 코딩 실습
   □ Turbine tree 시뮬레이션

3. AccountsDB
   □ runtime/src/accounts_db.rs 읽기
   □ AppendVec 구조 이해
   □ 압축 과정 분석

4. Banking Stage
   □ core/src/banking_stage.rs
   □ 병렬 실행 메커니즘
   □ 계정 충돌 감지

실습 프로젝트:
□ PoH 검증기
□ Shred 시뮬레이터
□ 계정 변경 추적 도구
□ 간단한 Solana 프로그램 (Rust)

코드 읽기 목표:
- Rust 문법에 익숙해지기
- 비동기 코드 (async/await) 이해
- 파이프라인 아키텍처 파악
```

#### Week 7-8: Sui 심화
```
목표: Sui 객체 모델 및 Move 이해

학습 순서:
1. 객체 모델
   □ crates/sui-types/src/object.rs
   □ 소유권 타입 이해
   □ 객체 vs 계정 비교

2. Move 프로그래밍
   □ Move 언어 기초
   □ Sui Move 확장
   □ 간단한 컨트랙트 작성

3. Narwhal & Tusk
   □ narwhal/types/src/header.rs
   □ DAG 구조 이해
   □ 합의 프로세스

4. Storage & Indexing
   □ authority_store.rs 분석
   □ 객체 버전 관리
   □ 소유권 인덱스

실습 프로젝트:
□ Move 스마트 컨트랙트 (NFT, Coin)
□ 객체 소유권 추적 도구
□ DAG 시각화 도구

코드 읽기 목표:
- Move 언어 습득
- 객체 모델 완전 이해
- Narwhal 구조 파악
```

#### Week 9-10: 비교 및 심화
```
목표: 3개 블록체인 비교 분석

활동:
□ 아키텍처 비교 문서 작성
□ 성능 벤치마크 분석
□ 트레이드오프 이해

주제:
1. 합의 알고리즘 비교
   - Gasper vs Tower BFT vs Narwhal-Tusk
   - Finality 메커니즘
   - 보안 vs 성능

2. 상태 관리 비교
   - Trie vs Flat vs Object
   - 병렬 실행 가능성
   - 스토리지 효율성

3. 네트워크 비교
   - 전파 메커니즘
   - 대역폭 효율성
   - 레이턴시

실습:
□ 각 블록체인에 같은 앱 배포
□ 성능 비교
□ 개발 경험 비교

산출물:
- 비교 분석 블로그 포스트
- 아키텍처 다이어그램
- 의사 결정 가이드
```

#### Week 11-12: 종합 프로젝트
```
목표: 실전 프로젝트로 지식 통합

프로젝트 아이디어:
1. Multi-chain 지갑
   - Ethereum, Solana, Sui 지원
   - 각 체인의 RPC 사용
   - 트랜잭션 서명 및 전송

2. Cross-chain 브리지 프로토타입
   - 메시지 검증 메커니즘
   - 락/민트 패턴
   - 이벤트 모니터링

3. 블록체인 분석 도구
   - 실시간 트랜잭션 모니터링
   - 성능 메트릭 수집
   - 시각화 대시보드

4. 커스텀 블록체인
   - 3개 체인의 장점 결합
   - 특정 use case 최적화
   - 프로토타입 구현

산출물:
- 작동하는 프로젝트
- 기술 문서
- 발표 자료
- GitHub 오픈소스
```

### 4.2 학습 우선순위 (개인 맞춤)

#### 백엔드 개발자
```
우선순위:
1. Ethereum (가장 안정적, 많은 자료)
2. Solana (고성능, Rust)
3. Sui (최신 기술)

중점:
- RPC 서버 구조
- 데이터베이스 설계
- 상태 관리
- API 설계

스킵 가능:
- 암호학 세부사항
- 합의 알고리즘 증명
- EVM/VM 내부
```

#### 시스템 프로그래머
```
우선순위:
1. Solana (복잡한 파이프라인)
2. Ethereum (안정적 구조)
3. Sui (최신 아키텍처)

중점:
- 네트워크 프로토콜
- 병렬 처리
- 메모리 관리
- 성능 최적화

깊이 학습:
- P2P 네트워킹
- 합의 알고리즘
- 데이터 구조
```

#### 블록체인 연구자
```
우선순위:
1. 모두 동등하게

중점:
- 합의 알고리즘 이론
- 보안 분석
- 성능 모델링
- 트레이드오프 분석

깊이 학습:
- 암호학 기초
- 분산 시스템
- 게임 이론
- 논문 읽기
```

#### 스마트 컨트랙트 개발자
```
우선순위:
1. Ethereum (가장 큰 생태계)
2. Sui (안전한 Move)
3. Solana (성능)

중점:
- 컨트랙트 언어 (Solidity, Move, Rust)
- 보안 패턴
- Gas 최적화
- 테스팅

실습:
- DeFi 프로토콜
- NFT 마켓플레이스
- DAO
```

---

## 5. 실습 프로젝트

### 5.1 난이도별 프로젝트

#### 초급 (Week 1-4)

**프로젝트 1: 간단한 블록체인 익스플로러**
```
기술 스택:
- Frontend: React + Web3.js/ethers.js
- Backend: Node.js + RPC

기능:
□ 최신 블록 조회
□ 트랜잭션 검색
□ 주소 잔액 조회
□ 블록 상세 정보

학습 목표:
- RPC API 이해
- 블록 구조 이해
- 트랜잭션 해독
```

**프로젝트 2: Merkle Tree 라이브러리**
```
언어: Python 또는 Go

기능:
□ Merkle tree 생성
□ Merkle proof 생성
□ Proof 검증
□ 시각화

학습 목표:
- 해시 함수 이해
- 트리 구조
- 암호학적 검증
```

#### 중급 (Week 5-8)

**프로젝트 3: 이벤트 모니터링 봇**
```
기술 스택:
- Ethereum: ethers.js
- Solana: @solana/web3.js
- Sui: @mysten/sui.js

기능:
□ 특정 이벤트 구독 (WebSocket)
□ 필터링 및 알림
□ 데이터베이스 저장
□ Telegram/Discord 알림

학습 목표:
- WebSocket 구독
- 이벤트 파싱
- 실시간 처리
```

**프로젝트 4: 간단한 DEX (각 체인)**
```
Ethereum: Solidity
Solana: Anchor (Rust)
Sui: Move

기능:
□ 토큰 스왑
□ 유동성 풀
□ 가격 결정 (AMM)

학습 목표:
- 스마트 컨트랙트 개발
- 보안 고려사항
- 테스팅
- 각 체인의 차이점
```

#### 고급 (Week 9-12)

**프로젝트 5: Multi-chain 지갑**
```
기술 스택:
- Frontend: React Native
- Backend: Node.js
- 3개 체인 SDK

기능:
□ 키 관리 (HD Wallet)
□ 트랜잭션 서명
□ 잔액 조회 (3개 체인)
□ 트랜잭션 전송
□ 히스토리

학습 목표:
- 키 관리
- 서명 알고리즘
- 각 체인의 트랜잭션 구조
- 보안
```

**프로젝트 6: 블록체인 분석 대시보드**
```
기술 스택:
- Backend: Python (데이터 수집)
- Database: TimescaleDB
- Frontend: Grafana / React

기능:
□ TPS 실시간 모니터링
□ Gas price 추적
□ 네트워크 상태
□ 검증자 성능
□ 비교 차트 (3개 체인)

학습 목표:
- 메트릭 수집
- 시계열 데이터
- 시각화
- 성능 분석
```

**프로젝트 7: 커스텀 Layer 2**
```
기술 스택:
- Rust 또는 Go
- Ethereum L1 (settlement)

기능:
□ 트랜잭션 배치 처리
□ State root 커밋
□ Fraud proof (Optimistic) 또는 Validity proof (ZK)
□ 브리지

학습 목표:
- L2 아키텍처
- Rollup 메커니즘
- 보안 모델
- 실제 구현
```

### 5.2 프로젝트별 학습 자원

**Ethereum 프로젝트**
```
문서:
- ethereum.org/developers
- docs.soliditylang.org

도구:
- Hardhat: hardhat.org
- Foundry: book.getfoundry.sh
- Tenderly: tenderly.co (디버깅)

튜토리얼:
- CryptoZombies
- Solidity by Example
- Scaffold-ETH

코드 예제:
- OpenZeppelin Contracts
- Uniswap V2/V3
```

**Solana 프로젝트**
```
문서:
- docs.solana.com
- solanacookbook.com

도구:
- Anchor: anchor-lang.com
- Solana CLI
- Solana Explorer

튜토리얼:
- Anchor Book
- Solana Bootcamp
- paulx.dev

코드 예제:
- SPL Token
- Metaplex
- Serum DEX
```

**Sui 프로젝트**
```
문서:
- docs.sui.io
- move-book.com

도구:
- Sui CLI
- Sui Explorer
- Sui Wallet

튜토리얼:
- Sui Move by Example
- examples.sui.io

코드 예제:
- sui-framework
- Sui Move examples
```

---

## 6. 코드 읽기 기법

### 6.1 Top-Down vs Bottom-Up

**Top-Down 접근 (추천: 초보자)**
```
1. 전체 아키텍처 파악
   - README, 문서 읽기
   - 디렉토리 구조 이해
   - 주요 컴포넌트 식별

2. 데이터 흐름 추적
   - 진입점 찾기 (main.go, lib.rs)
   - 주요 경로 따라가기
   - 호출 그래프 그리기

3. 세부 구현
   - 핵심 함수 분석
   - 알고리즘 이해
   - 최적화 기법

예: Ethereum 트랜잭션 처리
main() -> eth.New() -> eth.handler.Start() -> handleMsg() ->
core.InsertChain() -> statedb.Commit() -> trie.Commit()
```

**Bottom-Up 접근 (추천: 경험자)**
```
1. 기본 데이터 구조
   - types/ 디렉토리 먼저
   - 핵심 타입 정의 이해

2. 알고리즘 및 유틸리티
   - 순수 함수부터
   - 의존성 없는 모듈

3. 통합
   - 어떻게 조합되는지
   - 전체 시스템 이해

예: Solana Shred 이해
Shred struct -> ShredType enum -> Shredder::new() ->
entries_to_shreds() -> broadcast_stage -> turbine
```

### 6.2 효과적인 주석 달기

**좋은 주석 예시 (Ethereum StateDB)**
```go
// StateDB: Ethereum 월드 스테이트 관리
// 역할: 모든 계정 및 스토리지 상태 관리
// 패턴: Copy-on-Write, Journal for rollback
type StateDB struct {
    // 백엔드 트라이 데이터베이스
    // 목적: 영구 저장 및 Merkle proof 생성
    db   Database

    // 메인 계정 트라이
    // 구조: address (Keccak256) -> RLP(account)
    trie Trie

    // 계정 상태 캐시
    // 목적: 중복 트라이 조회 방지, 성능 10x 향상
    // 주의: 메모리 사용량 증가
    stateObjects      map[common.Address]*stateObject

    // 수정된 계정들 (커밋 대상)
    // 사용: Commit() 시점에 순회하여 트라이에 기록
    stateObjectsDirty map[common.Address]struct{}

    // ... 기타 필드들
}

// GetOrNewStateObject: 계정 조회 또는 생성
// 흐름:
// 1. 캐시 확인 (O(1))
// 2. 트라이 조회 (O(log n))
// 3. 없으면 빈 계정 생성
// 4. 캐시에 저장 후 반환
func (s *StateDB) GetOrNewStateObject(addr common.Address) *stateObject {
    // ...
}
```

**주석 작성 템플릿**
```
파일 상단:
// <파일명>: <주요 목적>
// 위치: <패키지 경로>
// 의존성: <주요 의존 모듈>
// 핵심 컨셉: <이해해야 할 개념들>

구조체:
// <이름>: <한 줄 설명>
// 역할: <이 구조체가 하는 일>
// 패턴: <사용하는 디자인 패턴>
// 주의사항: <함정, 제약사항>

함수:
// <이름>: <한 줄 설명>
// 흐름: <1, 2, 3 단계>
// 시간 복잡도: O(?)
// 공간 복잡도: O(?)
// 부작용: <상태 변경, 락 획득 등>
```

### 6.3 코드 읽기 도구

**IDE 설정**
```
VSCode 추천 익스텐션:
- Go: Go (official)
- Rust: rust-analyzer
- Move: move-analyzer

기능 활용:
- Go to Definition (F12)
- Find All References (Shift+F12)
- Call Hierarchy (Shift+Alt+H)
- Outline (Ctrl+Shift+O)

팁:
- 브레드크럼 활용
- 미니맵으로 전체 구조 파악
- 북마크 기능으로 중요 위치 표시
```

**정적 분석 도구**
```bash
# Go (Ethereum)
go doc <package>.<type>.<method>
go mod graph  # 의존성 그래프
goimports -w .  # Import 정리

# Rust (Solana, Sui)
cargo doc --open  # 문서 생성 및 열기
cargo tree  # 의존성 트리
cargo clippy  # Lint

# 호출 그래프 생성
go get -u github.com/ofabry/go-callvis
go-callvis -group pkg,type <package>
```

**다이어그램 도구**
```
- draw.io (diagrams.net): 아키텍처 다이어그램
- PlantUML: 텍스트로 UML
- Mermaid: 마크다운 다이어그램
- Graphviz: 그래프 시각화

예: 데이터 흐름도
graph TD
    A[User] -->|tx| B[RPC]
    B --> C[TxPool]
    C --> D[Miner]
    D --> E[EVM]
    E --> F[StateDB]
```

---

## 7. 추가 학습 자원

### 7.1 필수 논문

**Ethereum**
- Ethereum Whitepaper (Vitalik Buterin, 2013)
- Ethereum Yellow Paper (Gavin Wood, 2014)
- Casper FFG (Buterin & Griffith, 2017)
- EIP-1559: Fee Market Change

**Solana**
- Solana: A new architecture for a high performance blockchain (Yakovenko, 2017)
- Proof of History: A Clock for Blockchain
- Tower BFT

**Sui**
- Narwhal and Tusk: A DAG-based Mempool and Efficient BFT Consensus (Danezis et al., 2021)
- Sui: A Smart Contract Platform with High Throughput and Low Latency (Mysten Labs, 2022)
- FastPay: High-Performance Byzantine Fault Tolerant Settlement

### 7.2 블로그 & 비디오

**Ethereum**
- ethereum.org/developers
- Week in Ethereum News
- EthHub
- Finematics (YouTube)

**Solana**
- solana.com/news
- Solana Podcast
- anatoly.com (Yakovenko's blog)

**Sui**
- blog.sui.io
- Mysten Labs blog
- Sui Dev Portal

### 7.3 커뮤니티

**Discord & Forums**
- Ethereum: EthResear.ch, /r/ethereum
- Solana: Solana Discord, /r/solana
- Sui: Sui Discord, /r/sui

**GitHub Discussions**
- ethereum/go-ethereum
- solana-labs/solana (현재 anza-xyz/agave)
- MystenLabs/sui

---

## 마무리

### 핵심 요약

**Ethereum**: 안정적, 성숙함, 큰 생태계
- 학습: 중간 난이도
- 추천: 블록체인 입문자, 스마트 컨트랙트 개발자

**Solana**: 고성능, 복잡함, 성장 중
- 학습: 높은 난이도 (Rust 필수)
- 추천: 시스템 프로그래머, 성능 중시

**Sui**: 혁신적, 초기 단계, 안전함
- 학습: 높은 난이도 (새로운 개념)
- 추천: 연구자, 최신 기술 관심자

### 최종 조언

```
1. 한 번에 하나씩
   - 3개를 동시에 배우지 말 것
   - Ethereum 먼저 추천

2. 코드 직접 실행
   - 읽기만 하지 말고 실행하고 수정하기
   - 로컬 네트워크 구성 필수

3. 작은 프로젝트부터
   - 거창한 프로젝트보다 작은 것 여러 개
   - 매주 1개 완성 목표

4. 커뮤니티 참여
   - 질문하고 답변하기
   - 오픈소스 기여

5. 인내심
   - 복잡한 시스템, 시간 필요
   - 12주는 최소, 평생 학습
```

**행운을 빕니다! 🚀**
