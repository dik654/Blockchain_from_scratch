# 블록체인 학습 체크리스트

이 체크리스트는 12주 학습 로드맵을 따라 진행 상황을 추적하는 데 도움을 줍니다.

## 📋 사용 방법

- [ ] 체크박스를 사용하여 완료한 항목 표시
- 각 주차마다 권장 학습 시간: 10-15시간
- 막히는 부분은 건너뛰고 나중에 다시 시도

---

## Week 1-2: 기초 개념

### 블록체인 기본

- [ ] 블록체인 정의 이해
- [ ] 블록, 트랜잭션, 해시 개념
- [ ] 분산 합의의 필요성 이해
- [ ] Byzantine Generals Problem

**리소스**:
- [ ] Bitcoin whitepaper 읽기
- [ ] Ethereum whitepaper 읽기

### P2P 네트워킹

- [ ] P2P 네트워크 기본 개념
- [ ] TCP vs UDP 차이
- [ ] NAT traversal 이해
- [ ] DHT (Distributed Hash Table) 기본

### 암호학 기초

- [ ] 해시 함수 (SHA-256, Keccak-256)
- [ ] 공개키 암호화 (ECDSA, Ed25519)
- [ ] 디지털 서명
- [ ] Merkle Tree 원리

**실습**:
- [ ] `examples/ethereum/simple-merkle-tree.go` 실행
- [ ] Merkle proof 생성 및 검증
- [ ] 해시 함수 직접 구현 (SHA-256)

### 데이터베이스 기초

- [ ] 키-값 스토어 개념
- [ ] LevelDB vs RocksDB
- [ ] Write-Ahead Log (WAL)
- [ ] Compaction 이해

**미니 프로젝트**:
- [ ] 간단한 블록체인 구현 (Python/Go)
- [ ] 해시 체인 만들기
- [ ] Merkle Tree 라이브러리

---

## Week 3-4: Ethereum 심화

### 네트워크 레이어

**DevP2P**:
- [ ] `ETHEREUM_INTERNALS_SPEC.md` Section 1 읽기
- [ ] `p2p/server.go` 코드 읽기
- [ ] Server 구조체 이해
- [ ] Peer 관리 메커니즘

**RLPx**:
- [ ] `p2p/rlpx/rlpx.go` 분석
- [ ] ECIES 암호화 이해
- [ ] 핸드셰이크 프로세스
- [ ] 메시지 프레이밍

**노드 발견**:
- [ ] `p2p/discover/` 디렉토리 탐색
- [ ] Kademlia DHT 알고리즘
- [ ] ENR (Ethereum Node Records)
- [ ] Discovery v4 vs v5

**실습**:
- [ ] 로컬에서 2개 geth 노드 연결
- [ ] P2P 메시지 로깅
- [ ] 노드 발견 과정 관찰

### Merkle Patricia Trie

**기본 구조**:
- [ ] `trie/trie.go` 읽기
- [ ] fullNode, shortNode, hashNode 이해
- [ ] 경로 압축 메커니즘
- [ ] 해시 계산 과정

**Secure Trie**:
- [ ] `trie/secure_trie.go` 분석
- [ ] Keccak256 키 해싱
- [ ] Preimage 캐싱

**실습**:
- [ ] 간단한 Trie 직접 구현
- [ ] 삽입/조회 연산
- [ ] 해시 계산 추적
- [ ] Merkle proof 생성

### StateDB

**상태 관리**:
- [ ] `core/state/statedb.go` 읽기
- [ ] StateDB 구조체 이해
- [ ] stateObject 개념
- [ ] Copy-on-Write 메커니즘

**트랜잭션 실행**:
- [ ] `core/state_transition.go` 분석
- [ ] Gas 계산
- [ ] EVM 호출
- [ ] 상태 변경 추적

**캐싱 & 스냅샷**:
- [ ] `core/state/snapshot/` 탐색
- [ ] Snapshot 생성 과정
- [ ] Disk layer vs Diff layer
- [ ] 성능 최적화

**실습**:
- [ ] 트랜잭션 실행 과정 추적
- [ ] 상태 변경 시각화 도구
- [ ] Gas 사용량 분석

### RPC

**서버 구조**:
- [ ] `rpc/server.go` 읽기
- [ ] JSON-RPC 2.0 프로토콜
- [ ] 서비스 등록 메커니즘
- [ ] 리플렉션 사용

**ETH Namespace**:
- [ ] `internal/ethapi/api.go` 분석
- [ ] eth_getBalance 구현
- [ ] eth_sendTransaction 구현
- [ ] eth_call 시뮬레이션

**필터 & 구독**:
- [ ] `eth/filters/` 디렉토리
- [ ] WebSocket 구독
- [ ] 이벤트 시스템
- [ ] Bloom filter 활용

**실습**:
- [ ] 커스텀 RPC 메서드 추가
- [ ] Web3.js로 RPC 호출
- [ ] 이벤트 리스너 구현
- [ ] 블록체인 익스플로러 (간단한 버전)

### 주차 마무리

**미니 프로젝트**:
- [ ] Ethereum 블록체인 익스플로러
  - [ ] 최신 블록 조회
  - [ ] 트랜잭션 검색
  - [ ] 계정 잔액 조회
  - [ ] 이벤트 필터링

**복습**:
- [ ] 트랜잭션이 블록이 되는 전 과정 설명할 수 있음
- [ ] Merkle Patricia Trie 원리 이해
- [ ] StateDB의 역할 설명 가능

---

## Week 5-6: Solana 심화

### Proof of History

**기본 개념**:
- [ ] `SOLANA_INTERNALS_SPEC.md` Section 5.1 읽기
- [ ] `poh/src/poh_recorder.rs` 분석
- [ ] SHA-256 해시 체인 이해
- [ ] Tick vs Entry 차이

**PoH 생성**:
- [ ] `examples/solana/simple-poh.rs` 실행
- [ ] PoH 해시 계산 과정
- [ ] 트랜잭션 믹싱
- [ ] Tick 생성 메커니즘

**검증**:
- [ ] PoH 체인 검증 알고리즘
- [ ] 순서 변조 감지
- [ ] 시간 증명 원리

**실습**:
- [ ] PoH 검증기 구현
- [ ] hashes_per_tick 변경 실험
- [ ] 성능 측정

### Gossip Protocol

**CRDS**:
- [ ] `gossip/src/cluster_info.rs` 읽기
- [ ] ClusterInfo 구조체
- [ ] CrdsValue 타입들
- [ ] ContactInfo 이해

**프로토콜**:
- [ ] PlumTree 알고리즘
- [ ] Push/Pull 메커니즘
- [ ] Prune 메시지
- [ ] Bloom filter 사용

**실습**:
- [ ] Gossip 네트워크 모니터
- [ ] 노드 정보 수집
- [ ] 전파 시뮬레이션

### Turbine

**Shred 구조**:
- [ ] `ledger/src/shred.rs` 분석
- [ ] ShredCommonHeader 이해
- [ ] Data vs Coding Shred
- [ ] FEC 세트

**Reed-Solomon**:
- [ ] Erasure coding 원리
- [ ] 67 data + 33 coding
- [ ] 복구 알고리즘
- [ ] 성능 트레이드오프

**Turbine Tree**:
- [ ] `core/src/broadcast_stage/` 탐색
- [ ] Multi-layer tree 구조
- [ ] DATA_PLANE_FANOUT (200)
- [ ] 스테이크 가중 전파

**실습**:
- [ ] Shred 생성기
- [ ] FEC 복구 시뮬레이터
- [ ] Turbine tree 시각화

### AccountsDB

**구조**:
- [ ] `runtime/src/accounts_db.rs` 읽기
- [ ] AppendVec 파일 구조
- [ ] AccountsIndex 메커니즘
- [ ] 메모리 맵 파일

**연산**:
- [ ] 계정 로드
- [ ] 계정 저장
- [ ] 압축 (Compaction)
- [ ] 스냅샷

**실습**:
- [ ] 계정 변경 추적 도구
- [ ] 스토리지 분석기
- [ ] 압축 시뮬레이션

### Banking Stage

**병렬 실행**:
- [ ] `core/src/banking_stage.rs` 분석
- [ ] 계정 충돌 감지
- [ ] 배치 생성
- [ ] Rayon 병렬 처리

**파이프라인**:
- [ ] FetchStage
- [ ] SigVerifyStage (GPU)
- [ ] BankingStage
- [ ] BroadcastStage

**실습**:
- [ ] 트랜잭션 시뮬레이터
- [ ] 병렬 실행 벤치마크
- [ ] 충돌 분석 도구

### 주차 마무리

**미니 프로젝트**:
- [ ] Solana 성능 모니터
  - [ ] TPS 측정
  - [ ] 슬롯 진행 추적
  - [ ] 검증자 성능
  - [ ] 네트워크 상태

**복습**:
- [ ] PoH가 시간 증명하는 원리 설명 가능
- [ ] Turbine의 효율성 이해
- [ ] AccountsDB의 병렬 접근 메커니즘

---

## Week 7-8: Sui 심화

### 객체 모델

**기본 개념**:
- [ ] `SUI_INTERNALS_SPEC.md` Section 3.1 읽기
- [ ] `crates/sui-types/src/object.rs` 분석
- [ ] Object 구조체 이해
- [ ] Owner 타입들

**소유권**:
- [ ] AddressOwner (소유 객체)
- [ ] Shared (공유 객체)
- [ ] ObjectOwner (객체 소유)
- [ ] Immutable (불변 객체)

**실습**:
- [ ] `examples/sui/simple-coin.move` 배포
- [ ] 객체 생성 및 전송
- [ ] 소유권 추적

### Move 프로그래밍

**언어 기초**:
- [ ] Move Book 읽기
- [ ] 리소스 타입 이해
- [ ] Ability (key, store, copy, drop)
- [ ] 제네릭

**Sui Move**:
- [ ] UID와 object::new()
- [ ] transfer 함수들
- [ ] tx_context 사용
- [ ] 이벤트 발생

**실습**:
- [ ] NFT 컨트랙트 작성
- [ ] Staking 메커니즘
- [ ] 간단한 DEX
- [ ] DAO 투표

### Narwhal & Tusk

**DAG 구조**:
- [ ] `narwhal/types/src/header.rs` 읽기
- [ ] Header 구조체
- [ ] Certificate 생성
- [ ] 부모-자식 관계

**Primary-Worker**:
- [ ] `narwhal/primary/src/primary.rs` 분석
- [ ] `narwhal/worker/src/worker.rs` 분석
- [ ] 배치 생성
- [ ] Vote 수집

**Tusk 합의**:
- [ ] `narwhal/consensus/src/tusk.rs` 탐색
- [ ] Anchor 선택
- [ ] DAG 순서화
- [ ] 커밋 프로세스

**실습**:
- [ ] DAG 시각화 도구
- [ ] 합의 시뮬레이터
- [ ] 성능 분석

### Storage

**AuthorityStore**:
- [ ] `crates/sui-core/src/authority/authority_store.rs` 읽기
- [ ] TypedStore 사용
- [ ] 객체 저장소
- [ ] 버전 관리

**인덱싱**:
- [ ] owner_index 구조
- [ ] 트랜잭션 인덱스
- [ ] 체크포인트 저장

**실습**:
- [ ] 객체 조회 도구
- [ ] 버전 히스토리 추적
- [ ] 스토리지 분석

### 병렬 실행

**FastPath**:
- [ ] 소유 객체만 사용
- [ ] 합의 불필요
- [ ] 즉시 실행
- [ ] 성능 이점

**ConsensusPath**:
- [ ] 공유 객체 포함
- [ ] Narwhal 통과
- [ ] 순서 보장
- [ ] 레이턴시 증가

**실습**:
- [ ] FastPath vs ConsensusPath 비교
- [ ] 병렬 실행 벤치마크
- [ ] 최적화 기법

### 주차 마무리

**미니 프로젝트**:
- [ ] Sui DeFi 프로토콜
  - [ ] Liquidity Pool
  - [ ] Token Swap
  - [ ] Yield Farming
  - [ ] 이벤트 모니터링

**복습**:
- [ ] 객체 모델의 장점 설명 가능
- [ ] FastPath 조건 이해
- [ ] Narwhal의 역할 설명

---

## Week 9-10: 비교 및 심화

### 아키텍처 비교

**네트워크**:
- [ ] DevP2P vs Gossip+Turbine vs Narwhal
- [ ] 전파 효율성 비교
- [ ] 레이턴시 분석
- [ ] 대역폭 사용량

**합의**:
- [ ] Gasper vs Tower BFT vs Narwhal-Tusk
- [ ] Finality 메커니즘
- [ ] 보안 모델
- [ ] 성능 트레이드오프

**상태 관리**:
- [ ] Trie vs Flat vs Object
- [ ] 병렬 실행 가능성
- [ ] 스토리지 효율성
- [ ] 읽기/쓰기 성능

**RPC**:
- [ ] API 설계 비교
- [ ] 구독 메커니즘
- [ ] 확장성
- [ ] 개발자 경험

### 성능 분석

**벤치마크**:
- [ ] TPS 측정 방법론
- [ ] 각 체인에서 동일 앱 배포
- [ ] 성능 비교 (TPS, Finality, 비용)
- [ ] 병목 지점 분석

**스케일링**:
- [ ] 수평 확장 (샤딩)
- [ ] 수직 확장 (하드웨어)
- [ ] Layer 2 솔루션
- [ ] 미래 로드맵

### 보안 분석

**공격 벡터**:
- [ ] 51% 공격
- [ ] Long-range 공격
- [ ] Eclipse 공격
- [ ] MEV (Miner Extractable Value)

**방어 메커니즘**:
- [ ] 각 체인의 보안 모델
- [ ] 슬래싱 메커니즘
- [ ] 검증자 인센티브
- [ ] 네트워크 분산성

### 실습 프로젝트

**Multi-chain 앱**:
- [ ] 3개 체인 모두에 배포
- [ ] 성능 비교
- [ ] 개발 경험 비교
- [ ] 비용 분석

**분석 도구**:
- [ ] Multi-chain 익스플로러
- [ ] 성능 대시보드
- [ ] 비교 차트
- [ ] 실시간 모니터링

---

## Week 11-12: 종합 프로젝트

### 프로젝트 선택

**옵션 1: Multi-chain 지갑**:
- [ ] 아키텍처 설계
- [ ] 키 관리 (HD Wallet)
- [ ] 3개 체인 통합
- [ ] 트랜잭션 서명
- [ ] UI/UX 구현
- [ ] 테스팅
- [ ] 문서 작성

**옵션 2: Cross-chain 브리지**:
- [ ] 브리지 프로토콜 설계
- [ ] 락/민트 메커니즘
- [ ] Relayer 구현
- [ ] 메시지 검증
- [ ] 보안 분석
- [ ] 테스트넷 배포

**옵션 3: 블록체인 분석 플랫폼**:
- [ ] 데이터 수집 파이프라인
- [ ] TimescaleDB 스키마
- [ ] 메트릭 정의
- [ ] 시각화 대시보드
- [ ] 실시간 알림
- [ ] API 제공

**옵션 4: 커스텀 블록체인**:
- [ ] 요구사항 정의
- [ ] 아키텍처 설계
- [ ] 합의 알고리즘 선택
- [ ] 네트워크 구현
- [ ] 상태 관리
- [ ] RPC 서버
- [ ] 테스트 및 벤치마크

### 프로젝트 실행

**Week 11**:
- [ ] 프로젝트 설정
- [ ] 핵심 기능 구현 (50%)
- [ ] 중간 발표 준비

**Week 12**:
- [ ] 핵심 기능 완성 (100%)
- [ ] 테스팅 및 디버깅
- [ ] 문서화
- [ ] 최종 발표

### 발표 자료

- [ ] 프로젝트 개요
- [ ] 아키텍처 설명
- [ ] 기술적 도전과제
- [ ] 해결 방법
- [ ] 데모
- [ ] 향후 개선 사항

---

## 추가 학습 자료

### 논문

**Ethereum**:
- [ ] Ethereum Whitepaper
- [ ] Ethereum Yellow Paper
- [ ] Casper FFG
- [ ] EIP-1559

**Solana**:
- [ ] Solana Whitepaper
- [ ] Proof of History
- [ ] Tower BFT
- [ ] Sealevel

**Sui**:
- [ ] Narwhal and Tusk
- [ ] FastPay
- [ ] Sui Whitepaper

### 오픈소스 기여

- [ ] go-ethereum issue/PR 리뷰
- [ ] Solana issue/PR 리뷰
- [ ] Sui issue/PR 리뷰
- [ ] 문서 개선 PR
- [ ] 버그 리포트
- [ ] 작은 기능 추가

### 커뮤니티

- [ ] Ethereum Discord/Forum 가입
- [ ] Solana Discord 가입
- [ ] Sui Discord 가입
- [ ] 주간 뉴스레터 구독
- [ ] 컨퍼런스 참석 (온라인)

---

## 최종 평가

### 이론 이해도

- [ ] 각 체인의 합의 알고리즘 설명 가능
- [ ] 네트워크 프로토콜 차이점 이해
- [ ] 상태 관리 메커니즘 비교 가능
- [ ] 보안 모델 설명 가능
- [ ] 성능 트레이드오프 이해

### 실습 능력

- [ ] 각 체인의 RPC 사용 가능
- [ ] 스마트 컨트랙트 작성 (Solidity, Rust, Move)
- [ ] 블록체인 데이터 분석
- [ ] 성능 벤치마킹
- [ ] 디버깅 및 최적화

### 코드 읽기

- [ ] go-ethereum 주요 파일 이해
- [ ] Solana 주요 파일 이해
- [ ] Sui 주요 파일 이해
- [ ] 데이터 흐름 추적 가능
- [ ] 새로운 기능 추가 가능

---

## 🎉 축하합니다!

모든 체크리스트를 완료했다면, 당신은 이제:

✅ 블록체인 내부 구조를 깊이 이해하는 전문가
✅ 3개 주요 블록체인의 차이점을 설명할 수 있는 사람
✅ 실제 코드를 읽고 수정할 수 있는 개발자
✅ 블록체인 프로젝트에 기여할 수 있는 기술자

**다음 단계**:
- 오픈소스 프로젝트 기여
- 자신만의 블록체인/DApp 개발
- 블록체인 회사 지원
- 연구 논문 작성

**행운을 빕니다! 🚀**
