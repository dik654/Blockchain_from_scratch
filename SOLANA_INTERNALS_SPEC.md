# Solana 내부 구현 완전 분석

## 목차
1. [네트워크 레이어 (Network Layer)](#1-네트워크-레이어)
2. [데이터베이스 레이어 (Database Layer)](#2-데이터베이스-레이어)
3. [스토리지 레이어 (Storage Layer)](#3-스토리지-레이어)
4. [RPC 레이어 (RPC Layer)](#4-rpc-레이어)
5. [핵심 아키텍처 컴포넌트](#5-핵심-아키텍처-컴포넌트)

---

## 1. 네트워크 레이어 (Network Layer)

### 1.1 Gossip Protocol (Control Plane)

**소스 위치**: `gossip/src/`

#### 1.1.1 핵심 개념
```rust
// Gossip: Solana의 제어 평면(Control Plane)
// 목적: 클러스터 메타데이터 전파
//   - 노드 연결 정보
//   - 투표 정보
//   - 원장 높이
//   - 블록체인 상태

// 특징:
// - PlumTree 알고리즘 기반 (Epidemic Broadcast Trees)
// - UDP 기반 통신
// - 낮은 레이턴시, 높은 가용성
```

#### 1.1.2 ClusterInfo 구조
```rust
// ClusterInfo: 클러스터의 모든 노드 정보 관리
// 소스: gossip/src/cluster_info.rs

pub struct ClusterInfo {
    // 로컬 노드 정보
    pub id: Pubkey,

    // Gossip 테이블 (Cluster Replicated Data Store)
    // 목적: 네트워크의 모든 노드 정보 저장
    gossip: CrdsGossip,

    // 내부 로직:
    // - CRDT (Conflict-free Replicated Data Type) 사용
    // - 결과적 일관성(Eventual Consistency)
    // - 벡터 클락으로 버전 관리
}

pub struct CrdsGossip {
    // CRDS = Cluster Replicated Data Store
    // 목적: 분산 데이터 저장소

    crds: Crds,              // 실제 데이터
    push: CrdsGossipPush,    // Push 프로토콜
    pull: CrdsGossipPull,    // Pull 프로토콜
}

// CRDS 데이터 타입들
pub enum CrdsValue {
    // 1. ContactInfo: 노드 연결 정보
    ContactInfo(ContactInfo),
    // - Gossip 주소 (UDP)
    // - TPU 주소 (Transaction Processing Unit)
    // - TVU 주소 (Transaction Validation Unit)
    // - RPC 주소

    // 2. Vote: 투표 정보
    Vote(Vote),
    // - 검증자의 블록 투표

    // 3. LowestSlot: 가장 낮은 슬롯 번호
    LowestSlot(LowestSlot),
    // - 노드가 가진 가장 오래된 데이터

    // 4. SnapshotHashes: 스냅샷 해시
    SnapshotHashes(SnapshotHashes),
    // - 사용 가능한 스냅샷 정보

    // 5. AccountsHashes: 계정 해시
    AccountsHashes(AccountsHashes),

    // ... 기타 타입들
}

// ContactInfo 상세
pub struct ContactInfo {
    id: Pubkey,              // 노드 공개키

    // 네트워크 주소들
    gossip: SocketAddr,      // Gossip 수신 주소
    tvu: SocketAddr,         // Transaction Validation Unit
    tpu: SocketAddr,         // Transaction Processing Unit
    tpu_forwards: SocketAddr, // TPU forwarding
    repair: SocketAddr,      // Repair 서비스
    rpc: SocketAddr,         // JSON-RPC
    rpc_pubsub: SocketAddr,  // PubSub
    serve_repair: SocketAddr,

    // 벡터 클락 (버전 관리)
    wallclock: u64,          // 타임스탬프
    shred_version: u16,      // Shred 버전 (호환성)
}
```

#### 1.1.3 Gossip 프로토콜 메시지

```rust
// 5가지 메시지 타입
pub enum Protocol {
    // 1. PullRequest: 데이터 요청
    PullRequest(Bloom<Hash>, CrdsValue),
    // 의도:
    // - Bloom filter로 이미 가진 데이터 표시
    // - 서버는 Bloom에 없는 데이터만 응답

    // 2. PullResponse: Pull 응답
    PullResponse(Pubkey, Vec<CrdsValue>),
    // 의도: 요청한 노드에게 데이터 전송

    // 3. PushMessage: 새 데이터 전파
    PushMessage(Pubkey, Vec<CrdsValue>),
    // 의도:
    // - 새로운 정보를 이웃 노드들에게 능동적으로 전송
    // - Epidemic 방식 전파

    // 4. PruneMessage: 연결 가지치기
    PruneMessage(Pubkey, PruneData),
    // 의도:
    // - 중복 경로 제거
    // - 트래픽 최적화

    // 5. PingMessage / PongMessage: 생존 확인
    PingMessage(Ping),
    PongMessage(Pong),
}

// Gossip 루프 (매 100ms)
impl ClusterInfo {
    pub fn gossip_loop() {
        loop {
            // === PUSH 단계 ===
            // 의도: 새 정보를 능동적으로 전파

            // 1. 최근 업데이트된 값들 선택
            let new_values = self.crds.get_new_values();

            // 2. FANOUT (기본 6개) 이웃 노드 선택
            let push_peers = self.select_push_peers(PUSH_FANOUT);

            // 3. PushMessage 전송
            for peer in push_peers {
                send_udp(peer, PushMessage(new_values));
            }

            // === PULL 단계 ===
            // 의도: 놓친 정보를 능동적으로 요청

            // 4. Bloom filter 생성 (이미 가진 데이터)
            let bloom = self.crds.build_bloom_filter();

            // 5. 랜덤 노드들에게 PullRequest
            let pull_peers = self.select_random_peers(PULL_COUNT);
            for peer in pull_peers {
                send_udp(peer, PullRequest(bloom, self.my_contact_info));
            }

            // 6. 100ms 대기
            sleep(Duration::from_millis(100));
        }
    }

    // Push 메시지 수신 처리
    pub fn handle_push_message(&mut self, from: Pubkey, values: Vec<CrdsValue>) {
        // 1. 각 값에 대해
        for value in values {
            // 2. CRDS에 삽입 시도
            let is_new = self.crds.insert(value, timestamp);

            // 3. 새로운 정보면 다른 노드들에게 재전파
            if is_new {
                self.retransmit_queue.push(value);
            }
        }

        // 4. 중복 경로 감지시 Prune 메시지 전송
        if detect_duplicate_path() {
            send_udp(from, PruneMessage(...));
        }
    }

    // Pull 요청 처리
    pub fn handle_pull_request(&self, bloom: Bloom, from: ContactInfo) {
        // 1. Bloom filter에 없는 값들 필터링
        let missing_values: Vec<_> = self.crds.values()
            .filter(|v| !bloom.contains(v.hash()))
            .collect();

        // 2. PullResponse 전송
        send_udp(from.gossip, PullResponse(self.id, missing_values));
    }
}
```

### 1.2 Turbine (Data Plane)

**소스 위치**: `core/src/broadcast_stage/`, `ledger/src/shred.rs`

#### 1.2.1 핵심 개념
```rust
// Turbine: Solana의 블록 전파 메커니즘
// 목적: 블록 데이터를 전체 네트워크에 빠르게 전송

// 특징:
// - Multi-layer tree 구조 (BitTorrent 영감)
// - Reed-Solomon erasure coding
// - UDP 기반
// - 2-3 hop으로 전체 네트워크 도달

// 용어:
// - Shred: 블록의 조각 (기본 전송 단위)
// - Leader: 현재 블록을 생성하는 검증자
// - Layer: 전파 계층
```

#### 1.2.2 Shred 구조

```rust
// Shred: 블록 데이터의 원자 단위
// 소스: ledger/src/shred.rs

pub struct Shred {
    // 공통 헤더
    common_header: ShredCommonHeader,

    // Shred 타입별 헤더
    data_header: Option<DataShredHeader>,      // 데이터 Shred
    coding_header: Option<CodingShredHeader>,  // 코딩 Shred (FEC)

    // 페이로드
    payload: Vec<u8>,
}

pub struct ShredCommonHeader {
    signature: Signature,      // 리더의 서명
    shred_variant: ShredType,  // Data or Coding
    slot: Slot,                // 슬롯 번호
    index: u32,                // Shred 인덱스
    version: u16,              // Shred 버전
    fec_set_index: u32,        // FEC 세트 인덱스
}

pub struct DataShredHeader {
    parent_offset: u16,        // 부모 슬롯 오프셋
    flags: ShredFlags,
    size: u16,                 // 데이터 크기
}

pub struct CodingShredHeader {
    num_data_shreds: u16,      // 데이터 Shred 개수
    num_coding_shreds: u16,    // 코딩 Shred 개수
    position: u16,             // 코딩 Shred 위치
}

// Shred 생성
impl Shredder {
    pub fn entries_to_shreds(
        &self,
        entries: &[Entry],         // 트랜잭션 엔트리들
        is_last_in_slot: bool,
        next_shred_index: u32,
    ) -> (Vec<Shred>, Vec<Shred>) {  // (data_shreds, coding_shreds)

        // === 1단계: Data Shreds 생성 ===
        // 의도: 실제 블록 데이터를 Shred 크기(~1200 bytes)로 분할

        let mut data_shreds = Vec::new();
        let mut serialized = bincode::serialize(entries)?;

        // 1200바이트씩 분할
        for chunk in serialized.chunks(PAYLOAD_SIZE) {
            let shred = Shred::new_from_data(
                self.slot,
                next_shred_index,
                chunk,
                is_last_in_slot,
            );

            // 리더 서명
            shred.sign(&self.keypair);
            data_shreds.push(shred);
            next_shred_index += 1;
        }

        // === 2단계: Coding Shreds 생성 (FEC) ===
        // 의도: Reed-Solomon erasure coding으로 복구 가능한 패리티 생성

        // FEC 설정: 67 data shreds -> 33 coding shreds
        // 결과: 100개 중 67개만 받아도 복구 가능!
        let coding_shreds = self.generate_coding_shreds(
            &data_shreds,
            NUM_CODING,  // 33
        );

        (data_shreds, coding_shreds)
    }

    fn generate_coding_shreds(
        &self,
        data_shreds: &[Shred],
        num_coding: usize,
    ) -> Vec<Shred> {
        // Reed-Solomon 인코더
        let encoder = ReedSolomon::new(
            data_shreds.len(),  // k = 67
            num_coding,         // m = 33
        ).unwrap();

        // 패리티 샤드 생성
        let mut shards = data_shreds.iter()
            .map(|s| s.payload.clone())
            .collect::<Vec<_>>();

        // 코딩 샤드 공간 할당
        for _ in 0..num_coding {
            shards.push(vec![0u8; PAYLOAD_SIZE]);
        }

        // 인코딩
        encoder.encode(&mut shards)?;

        // 코딩 Shred 객체 생성
        shards[data_shreds.len()..]
            .iter()
            .enumerate()
            .map(|(i, payload)| {
                Shred::new_coding_from_payload(
                    self.slot,
                    i as u16,
                    payload,
                    data_shreds.len(),
                    num_coding,
                )
            })
            .collect()
    }
}
```

#### 1.2.3 Turbine Tree 전파

```rust
// BroadcastStage: 리더가 Shred를 전파하는 단계
// 소스: core/src/broadcast_stage/broadcast_stage.rs

pub struct BroadcastStage {
    // 목적: 생성된 Shred들을 네트워크에 전파
}

impl BroadcastStage {
    pub fn broadcast_shreds(
        &self,
        shreds: Vec<Shred>,
        cluster_info: &ClusterInfo,
    ) {
        // === Turbine Tree 구성 ===
        // 목적: 효율적인 멀티레이어 전파

        const DATA_PLANE_FANOUT: usize = 200;  // 각 노드의 자식 수

        // 1. 활성 검증자 목록 (스테이크 가중치)
        let mut peers: Vec<_> = cluster_info
            .all_tvu_peers()  // TVU = Transaction Validation Unit
            .into_iter()
            .collect();

        // 2. 스테이크 순으로 정렬
        // 의도: 높은 스테이크 노드가 먼저 받도록 (중요!)
        peers.sort_by_key(|p| Reverse(p.stake));

        // 3. Turbine Tree 레이어 구성
        // Layer 0 (리더): fanout 200개 자식
        // Layer 1: 각각 fanout 200개 자식
        // Layer 2: ...

        // 예: 4만 노드 = 200 (L1) + 200*200 (L2) = 40,000
        //     -> 최대 2 hop으로 전체 도달!

        for (shred_index, shred) in shreds.iter().enumerate() {
            // 각 Shred마다 다른 노드 순서 (부하 분산)
            let index_seed = shred_index;
            let rotated_peers = rotate_peers(&peers, index_seed);

            // Layer 0: 리더 -> 첫 200개 노드
            let layer0_peers = &rotated_peers[..DATA_PLANE_FANOUT];

            for peer in layer0_peers {
                // UDP로 Shred 전송
                send_shred_udp(peer.tvu, shred);
            }
        }

        // 각 수신 노드는 자신의 자식들에게 재전파
        // (재귀적 전파)
    }
}

// Retransmit Stage: 수신한 Shred를 자식들에게 재전파
// 소스: core/src/retransmit_stage.rs

pub struct RetransmitStage {
    // 목적: Turbine Tree에서 중간 노드 역할
}

impl RetransmitStage {
    pub fn retransmit_shreds(&self, shred: Shred) {
        // 1. 자신의 위치 계산
        let my_index = self.calculate_my_index_in_tree();

        // 2. 자식 노드들 결정
        // 공식: children = [my_index * FANOUT + 1 .. my_index * FANOUT + FANOUT]
        let start = my_index * DATA_PLANE_FANOUT + 1;
        let end = start + DATA_PLANE_FANOUT;
        let children = &all_peers[start..end];

        // 3. 자식들에게 재전파
        for child in children {
            send_shred_udp(child.tvu, &shred);
        }

        // 결과:
        // - 200^2 = 40,000 노드를 2 hop에 커버
        // - 각 노드는 200개만 전송 (부하 분산!)
    }
}

// Shred 복구 (FEC)
pub struct Reconstructor {
    // 목적: 손실된 Shred를 코딩 Shred로 복구

    pub fn try_reconstruct(&mut self, slot: Slot) -> Result<Vec<Shred>> {
        // FEC 세트별로 수신된 Shred 확인
        let fec_set = self.incomplete_sets.get(slot)?;

        // 67개 중 일부만 받은 경우
        let data_shreds_count = fec_set.data_shreds.len();
        let coding_shreds_count = fec_set.coding_shreds.len();

        // 총 67개 이상 있으면 복구 가능
        if data_shreds_count + coding_shreds_count >= ORIGINAL_COUNT {
            // Reed-Solomon 디코딩
            let recovered = self.reed_solomon_decode(fec_set)?;

            // 복구된 Shred 반환
            Ok(recovered)
        } else {
            // 아직 부족 -> 더 기다림
            Err(NotEnoughShreds)
        }
    }
}
```

### 1.3 TVU & TPU

**소스 위치**: `core/src/tvu.rs`, `core/src/tpu.rs`

```rust
// TVU (Transaction Validation Unit)
// 목적: 블록 검증 파이프라인

pub struct Tvu {
    // 의도: 리더가 아닌 검증자가 블록을 검증

    // Shred 수신
    fetch_stage: FetchStage,           // Shred UDP 수신
    shred_fetch_stage: ShredFetchStage,

    // Shred 처리
    retransmit_stage: RetransmitStage, // 재전파
    repair_service: RepairService,      // 손실 Shred 요청

    // 블록 재구성 및 검증
    blockstore_processor: BlockstoreProcessor,

    // 합의 참여
    voting_service: VotingService,      // 투표 전송
}

// TPU (Transaction Processing Unit)
// 목적: 트랜잭션 처리 파이프라인 (리더만)

pub struct Tpu {
    // 의도: 리더가 새 블록 생성

    // 트랜잭션 수신
    fetch_stage: FetchStage,            // 트랜잭션 수신
    sigverify_stage: SigVerifyStage,    // 서명 검증 (GPU 가속)

    // 뱅킹
    banking_stage: BankingStage,        // 트랜잭션 실행

    // 블록 생성 및 전파
    broadcast_stage: BroadcastStage,    // Shred 전파
}
```

---

## 2. 데이터베이스 레이어 (Database Layer)

### 2.1 RocksDB 통합

**소스 위치**: `ledger/src/blockstore_db.rs`

#### 2.1.1 BlockstoreDB
```rust
// BlockstoreDB: RocksDB 래퍼
// 목적: Ledger 데이터 영구 저장

use rocksdb::{
    DB, ColumnFamily, Options, DBCompressionType,
    WriteBatch, IteratorMode,
};

pub struct BlockstoreDb {
    // RocksDB 인스턴스
    db: Arc<DB>,

    // Column Families (테이블들)
    // 목적: 데이터 타입별 분리 저장
}

// Column Family 정의
pub enum Column {
    // === Shred 저장 ===
    ShredData,        // 데이터 Shred
    ShredCode,        // 코딩 Shred (FEC)

    // === 슬롯 메타데이터 ===
    SlotMeta,         // 슬롯 정보 (부모, 자식, 완료 여부)
    DeadSlots,        // 포크된 슬롯들
    OrphanSlots,      // 고아 슬롯들

    // === 트랜잭션 인덱스 ===
    TransactionStatus,         // 트랜잭션 상태
    TransactionStatusIndex,    // 인덱스

    // === 블록 정보 ===
    Rewards,          // 블록 보상
    Blocktime,        // 블록 타임스탬프
    PerfSamples,      // 성능 샘플
    BlockHeight,      // 블록 높이

    // === 계정 관련 ===
    // (AccountsDB가 주로 사용, 여기는 보조)

    // ... 기타
}

// 키 스킴
impl BlockstoreDb {
    // Shred 키: slot (u64) + index (u32)
    fn shred_key(slot: Slot, index: u32) -> Vec<u8> {
        // 빅 엔디안 인코딩
        // 목적: 슬롯 순서대로 정렬
        let mut key = slot.to_be_bytes().to_vec();
        key.extend_from_slice(&index.to_be_bytes());
        key
    }

    // Shred 쓰기
    pub fn insert_shreds(&self, shreds: Vec<Shred>) -> Result<()> {
        // 배치 쓰기 (원자성)
        let mut batch = WriteBatch::default();

        for shred in shreds {
            let key = Self::shred_key(shred.slot(), shred.index());

            // 데이터 or 코딩 CF 선택
            let cf = if shred.is_data() {
                self.db.cf_handle(Column::ShredData)
            } else {
                self.db.cf_handle(Column::ShredCode)
            };

            // 배치에 추가
            batch.put_cf(cf, key, shred.payload());
        }

        // 원자적 커밋
        self.db.write(batch)?;
        Ok(())
    }

    // Shred 읽기
    pub fn get_data_shred(&self, slot: Slot, index: u32) -> Result<Option<Shred>> {
        let key = Self::shred_key(slot, index);
        let cf = self.db.cf_handle(Column::ShredData)?;

        match self.db.get_cf(cf, key)? {
            Some(bytes) => Ok(Some(Shred::from_payload(bytes)?)),
            None => Ok(None),
        }
    }

    // 슬롯의 모든 Shred 반복
    pub fn slot_data_iterator(&self, slot: Slot) -> impl Iterator<Item = Shred> {
        let start_key = Self::shred_key(slot, 0);
        let end_key = Self::shred_key(slot + 1, 0);

        let cf = self.db.cf_handle(Column::ShredData).unwrap();

        self.db
            .iterator_cf(cf, IteratorMode::From(&start_key, Direction::Forward))
            .take_while(move |(key, _)| key < &end_key)
            .map(|(_, value)| Shred::from_payload(value).unwrap())
    }
}
```

#### 2.1.2 SlotMeta

```rust
// SlotMeta: 슬롯 메타데이터
// 목적: 슬롯의 상태 및 관계 추적

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct SlotMeta {
    // 슬롯 번호
    pub slot: Slot,

    // 부모-자식 관계
    pub parent_slot: Option<Slot>,     // 부모 슬롯
    pub next_slots: Vec<Slot>,         // 자식 슬롯들 (포크 가능)

    // Shred 수신 상태
    pub consumed: u64,                 // 연속으로 받은 Shred 수
    pub received: u64,                 // 총 받은 Shred 수
    pub last_index: Option<u64>,       // 마지막 Shred 인덱스

    // 완료 플래그
    // 의도: 모든 Shred를 받았는지 확인
    pub is_full: bool,
    pub is_connected: bool,            // 제네시스까지 연결됨
}

impl BlockstoreDb {
    // SlotMeta 업데이트
    pub fn update_slot_meta(&self, slot: Slot, meta: SlotMeta) -> Result<()> {
        let key = slot.to_be_bytes();
        let value = bincode::serialize(&meta)?;

        let cf = self.db.cf_handle(Column::SlotMeta)?;
        self.db.put_cf(cf, key, value)?;

        Ok(())
    }

    // 체인 연결 여부 확인
    pub fn is_connected_to_root(&self, slot: Slot) -> bool {
        // 부모를 따라가며 루트까지 도달 가능한지 확인
        let mut current = slot;

        loop {
            let meta = self.get_slot_meta(current)?;

            if meta.is_rooted {
                return true;  // 루트에 연결됨
            }

            match meta.parent_slot {
                Some(parent) => current = parent,
                None => return false,  // 고아 슬롯
            }
        }
    }
}
```

---

## 3. 스토리지 레이어 (Storage Layer)

### 3.1 AccountsDB

**소스 위치**: `runtime/src/accounts_db.rs`

#### 3.1.1 핵심 개념
```rust
// AccountsDB: Solana의 계정 상태 저장소
// 목적: 수백만 계정의 효율적인 저장 및 빠른 조회

// 특징:
// - Append-only 스토리지
// - 메모리 맵 파일 (mmap)
// - Copy-on-Write (COW)
// - 백그라운드 압축 (Compaction)
// - 스냅샷 지원

pub struct AccountsDb {
    // === 저장소 ===
    // AppendVec의 맵
    // 목적: 계정 데이터를 슬롯별로 분리 저장
    pub storage: AccountStorage,

    // === 인덱스 ===
    // 계정 Pubkey -> (슬롯, AppendVec ID, 오프셋)
    // 목적: O(1) 계정 조회
    pub accounts_index: AccountsIndex<AccountInfo>,

    // === 캐시 ===
    // 최근 업데이트된 계정들 (메모리)
    pub accounts_cache: AccountsCache,

    // === 스냅샷 ===
    pub snapshot_storages: RwLock<Vec<SnapshotStorage>>,
}
```

#### 3.1.2 AppendVec (계정 파일)

```rust
// AppendVec: Append-only 계정 저장 파일
// 소스: runtime/src/append_vec.rs

pub struct AppendVec {
    // 메모리 맵 파일
    // 목적:
    // - 커널 페이지 캐시 활용
    // - 제로카피 I/O
    map: MmapMut,

    // 현재 쓰기 오프셋
    current_len: AtomicUsize,

    // 파일 경로: <slot>.<id>
    path: PathBuf,

    // 파일 크기 (고정, 예: 64MB)
    file_size: u64,
}

// 계정 저장 형식
#[repr(C)]
pub struct StoredAccountMeta {
    // 메타 정보
    write_version: u64,    // 쓰기 버전
    pubkey: Pubkey,        // 계정 공개키 (32 bytes)

    // 계정 데이터
    lamports: u64,         // 잔액
    owner: Pubkey,         // 소유자 프로그램
    executable: bool,      // 실행 가능 여부
    rent_epoch: Epoch,     // Rent 에포크

    // 데이터 길이 및 포인터
    data_len: u64,
    data: *const u8,       // 실제 데이터 (가변 길이)

    // 해시
    hash: Hash,
}

impl AppendVec {
    // 계정 추가 (append-only)
    pub fn append_account(
        &self,
        pubkey: &Pubkey,
        account: &Account,
        hash: Hash,
    ) -> Option<usize> {  // 반환: 오프셋
        // 1. 필요한 공간 계산
        let required_space =
            mem::size_of::<StoredAccountMeta>() + account.data.len();

        // 2. 현재 오프셋 확보 (원자적)
        let offset = self.current_len.fetch_add(required_space, Ordering::Relaxed);

        // 3. 공간 부족 확인
        if offset + required_space > self.file_size as usize {
            return None;  // 이 AppendVec는 꽉 참
        }

        // 4. 메모리 맵에 직접 쓰기
        unsafe {
            let ptr = self.map.as_mut_ptr().add(offset);

            // 메타 데이터 쓰기
            let stored = ptr as *mut StoredAccountMeta;
            (*stored).pubkey = *pubkey;
            (*stored).lamports = account.lamports;
            (*stored).owner = account.owner;
            (*stored).data_len = account.data.len() as u64;
            (*stored).hash = hash;

            // 데이터 복사
            let data_ptr = ptr.add(mem::size_of::<StoredAccountMeta>());
            std::ptr::copy_nonoverlapping(
                account.data.as_ptr(),
                data_ptr,
                account.data.len(),
            );
        }

        Some(offset)
    }

    // 계정 읽기
    pub fn get_account(&self, offset: usize) -> Option<&StoredAccountMeta> {
        // 메모리 맵에서 직접 읽기 (제로카피!)
        unsafe {
            let ptr = self.map.as_ptr().add(offset);
            Some(&*(ptr as *const StoredAccountMeta))
        }
    }
}
```

#### 3.1.3 AccountsIndex

```rust
// AccountsIndex: 계정 Pubkey -> 위치 매핑
// 소스: runtime/src/accounts_index.rs

pub struct AccountsIndex<T> {
    // 메인 인덱스: Pubkey -> AccountMapEntry
    // 목적: 빠른 계정 조회
    pub account_maps: LockHashMap<Pubkey, AccountMapEntry<T>>,

    // 슬롯 리스트
    // 목적: 같은 계정의 버전들 추적
    pub slots_list: RwLock<HashMap<Pubkey, Vec<(Slot, T)>>>,

    // 루트 슬롯들
    // 목적: 확정된 상태만 유지
    pub roots: RwLock<HashSet<Slot>>,
}

pub struct AccountMapEntry<T> {
    // 슬롯-값 쌍들 (시간 순)
    // 예: [(100, info1), (150, info2), (200, info3)]
    slot_list: Vec<(Slot, T)>,

    // ref_count: 참조 카운트
    ref_count: AtomicU64,
}

pub struct AccountInfo {
    // AppendVec 위치 정보
    store_id: AppendVecId,  // 어떤 파일
    offset: usize,           // 파일 내 오프셋

    // 메타 정보
    lamports: u64,
    executable: bool,
}

impl AccountsIndex<AccountInfo> {
    // 계정 조회
    pub fn get(
        &self,
        pubkey: &Pubkey,
        ancestors: &Ancestors,  // 조회할 슬롯 범위
    ) -> Option<AccountInfo> {
        // 1. 인덱스에서 엔트리 찾기
        let entry = self.account_maps.get(pubkey)?;

        // 2. 슬롯 리스트에서 가장 최근의 조상 찾기
        // ancestors: {100, 150, 200} (현재 포크의 슬롯들)
        // slot_list: [(100, info1), (120, info2), (200, info3)]
        // 결과: (200, info3) 선택 (가장 최근의 조상)

        entry.slot_list
            .iter()
            .rev()  // 역순 (최신부터)
            .find(|(slot, _)| ancestors.contains(slot))
            .map(|(_, info)| info.clone())
    }

    // 계정 업데이트
    pub fn upsert(
        &self,
        slot: Slot,
        pubkey: &Pubkey,
        account_info: AccountInfo,
    ) {
        // 1. 엔트리 찾기 또는 생성
        let mut entry = self.account_maps
            .entry(*pubkey)
            .or_insert_with(AccountMapEntry::default);

        // 2. 슬롯 리스트에 추가
        // (같은 슬롯이면 덮어쓰기)
        if let Some(existing) = entry.slot_list.iter_mut()
            .find(|(s, _)| *s == slot) {
            existing.1 = account_info;  // 업데이트
        } else {
            entry.slot_list.push((slot, account_info));  // 추가
        }

        // 3. 슬롯 순으로 정렬 유지
        entry.slot_list.sort_by_key(|(slot, _)| *slot);
    }

    // 클린업 (오래된 버전 제거)
    pub fn clean_rooted_entries(&self, rooted_slot: Slot) {
        // 목적: 루트된 슬롯 이전의 계정 버전 제거

        for entry in self.account_maps.values_mut() {
            // 루트 슬롯 이후만 유지
            entry.slot_list.retain(|(slot, _)| *slot >= rooted_slot);
        }
    }
}
```

#### 3.1.4 AccountsDB 연산

```rust
impl AccountsDb {
    // === 계정 로드 ===
    pub fn load(
        &self,
        ancestors: &Ancestors,
        pubkey: &Pubkey,
    ) -> Option<Account> {
        // 1. 캐시 확인 (최근 업데이트)
        if let Some(cached) = self.accounts_cache.load(pubkey) {
            return Some(cached);
        }

        // 2. 인덱스에서 위치 조회
        let account_info = self.accounts_index.get(pubkey, ancestors)?;

        // 3. AppendVec에서 로드
        let storage = self.storage.get(account_info.store_id)?;
        let stored = storage.accounts.get_account(account_info.offset)?;

        // 4. Account 객체 생성
        Some(Account {
            lamports: stored.lamports,
            data: stored.data.to_vec(),
            owner: stored.owner,
            executable: stored.executable,
            rent_epoch: stored.rent_epoch,
        })
    }

    // === 계정 저장 ===
    pub fn store(
        &self,
        slot: Slot,
        accounts: &[(&Pubkey, &Account)],
    ) {
        // 1. 현재 슬롯의 AppendVec 가져오기 (또는 생성)
        let mut storage = self.get_or_create_storage(slot);

        // 2. 각 계정 저장
        for (pubkey, account) in accounts {
            // 2.1. 계정 해시 계산
            let hash = Self::hash_account(slot, account);

            // 2.2. AppendVec에 추가
            let offset = storage.accounts.append_account(pubkey, account, hash);

            // 2.3. AppendVec 꽉 차면 새로 생성
            if offset.is_none() {
                storage = self.create_new_storage(slot);
                offset = storage.accounts.append_account(pubkey, account, hash);
            }

            let offset = offset.unwrap();

            // 2.4. 인덱스 업데이트
            let account_info = AccountInfo {
                store_id: storage.id,
                offset,
                lamports: account.lamports,
                executable: account.executable,
            };

            self.accounts_index.upsert(slot, pubkey, account_info);
        }
    }

    // === 압축 (Compaction) ===
    pub fn shrink_ancient_append_vecs(&self) {
        // 목적: 오래된 계정 파일들을 압축하여 공간 절약

        // 1. 오래된 AppendVec 찾기
        let old_storages: Vec<_> = self.storage.values()
            .filter(|s| s.slot < self.get_old_slot_threshold())
            .collect();

        for old_storage in old_storages {
            // 2. 살아있는 계정만 추출
            let alive_accounts: Vec<_> = old_storage.accounts
                .accounts()
                .filter(|account| {
                    // 인덱스에 여전히 이 위치를 가리키는지 확인
                    self.is_account_alive(&account.pubkey, old_storage.id)
                })
                .collect();

            // 3. 새 AppendVec에 재작성
            let new_storage = self.create_new_storage(old_storage.slot);
            for account in alive_accounts {
                new_storage.accounts.append_account(&account.pubkey, &account, account.hash);
            }

            // 4. 인덱스 업데이트
            // ...

            // 5. 오래된 파일 삭제
            self.remove_storage(old_storage.id);
        }
    }
}
```

### 3.2 Snapshots

**소스 위치**: `runtime/src/snapshot_utils.rs`

```rust
// 스냅샷: 특정 슬롯의 전체 계정 상태
// 목적: 빠른 부트스트랩, 재시작, 복제

pub struct SnapshotPackage {
    // 슬롯 번호
    slot: Slot,

    // 계정 저장소 파일들
    account_storages: Vec<PathBuf>,

    // 뱅크 상태 (시스템 계정, 스테이크, 투표 등)
    bank_snapshot: BankSnapshot,

    // 압축 파일 경로
    snapshot_archive: PathBuf,
}

impl SnapshotUtils {
    // 스냅샷 생성
    pub fn create_snapshot(
        bank: &Bank,
        snapshot_path: &Path,
    ) -> Result<()> {
        let slot = bank.slot();

        // 1. 계정 스토리지 동결
        // 의도: 일관된 상태 캡처
        let storages = bank.get_snapshot_storages();

        // 2. 계정 파일들을 스냅샷 디렉토리에 하드링크
        // 목적: 디스크 공간 절약 (복사 안함!)
        let snapshot_dir = snapshot_path.join(slot.to_string());
        fs::create_dir_all(&snapshot_dir)?;

        for storage in &storages {
            let src = &storage.path;
            let dst = snapshot_dir.join(src.file_name().unwrap());
            fs::hard_link(src, dst)?;  // 하드링크!
        }

        // 3. 뱅크 상태 직렬화
        let bank_snapshot = BankSnapshot::from_bank(bank);
        let bank_file = snapshot_dir.join("bank.bin");
        bincode::serialize_into(File::create(bank_file)?, &bank_snapshot)?;

        // 4. 상태 해시 파일
        let hash = bank.hash();
        let hash_file = snapshot_dir.join("hash.bin");
        bincode::serialize_into(File::create(hash_file)?, &hash)?;

        // 5. 압축 (tar.zst)
        Self::compress_snapshot(&snapshot_dir)?;

        Ok(())
    }

    // 스냅샷 로드
    pub fn load_snapshot(
        snapshot_archive: &Path,
        account_paths: &[PathBuf],
    ) -> Result<Bank> {
        // 1. 압축 해제
        let temp_dir = Self::decompress_snapshot(snapshot_archive)?;

        // 2. 뱅크 상태 역직렬화
        let bank_file = temp_dir.join("bank.bin");
        let bank_snapshot: BankSnapshot = bincode::deserialize_from(File::open(bank_file)?)?;

        // 3. 계정 파일들을 데이터 디렉토리로 복사/이동
        for storage_file in temp_dir.read_dir()? {
            if storage_file.path().extension() == Some("accounts") {
                let dst = account_paths[0].join(storage_file.file_name());
                fs::copy(storage_file.path(), dst)?;
            }
        }

        // 4. AccountsDB 재구성
        let accounts_db = AccountsDb::new(account_paths);
        accounts_db.load_from_snapshot(&bank_snapshot)?;

        // 5. Bank 재구성
        let bank = Bank::from_snapshot(bank_snapshot, accounts_db)?;

        Ok(bank)
    }
}
```

---

## 4. RPC 레이어 (RPC Layer)

### 4.1 JSON-RPC 서버

**소스 위치**: `rpc/src/rpc.rs`

#### 4.1.1 서버 구조
```rust
// JsonRpcService: JSON-RPC 서버
// 소스: rpc/src/rpc_service.rs

pub struct JsonRpcService {
    // HTTP 서버
    http_server: Option<HttpServer>,

    // WebSocket 서버 (PubSub)
    pubsub_server: Option<PubSubServer>,

    // 요청 핸들러
    request_processor: Arc<JsonRpcRequestProcessor>,
}

// JsonRpcRequestProcessor: 요청 처리
pub struct JsonRpcRequestProcessor {
    // 뱅크 (현재 상태)
    bank_forks: Arc<RwLock<BankForks>>,

    // 블록스토어 (히스토리)
    blockstore: Arc<Blockstore>,

    // 트랜잭션 전송
    tpu_address: SocketAddr,

    // 설정
    config: JsonRpcConfig,
}

// jsonrpc-core 사용
use jsonrpc_core::{IoHandler, Result as RpcResult};
use jsonrpc_derive::rpc;

// RPC 트레잇 정의
#[rpc(server)]
pub trait RpcSol {
    // 계정 정보 조회
    #[rpc(name = "getAccountInfo")]
    fn get_account_info(
        &self,
        pubkey: String,
        config: Option<RpcAccountInfoConfig>,
    ) -> RpcResult<RpcResponse<Option<UiAccount>>>;

    // 잔액 조회
    #[rpc(name = "getBalance")]
    fn get_balance(
        &self,
        pubkey: String,
        config: Option<RpcContextConfig>,
    ) -> RpcResult<RpcResponse<u64>>;

    // 블록 조회
    #[rpc(name = "getBlock")]
    fn get_block(
        &self,
        slot: Slot,
        config: Option<RpcEncodingConfigWrapper<RpcBlockConfig>>,
    ) -> RpcResult<Option<UiConfirmedBlock>>;

    // 트랜잭션 전송
    #[rpc(name = "sendTransaction")]
    fn send_transaction(
        &self,
        data: String,
        config: Option<RpcSendTransactionConfig>,
    ) -> RpcResult<String>;

    // ... 100+ 메서드들
}
```

#### 4.1.2 주요 RPC 메서드 구현

```rust
impl RpcSol for RpcSolImpl {
    // 계정 정보 조회
    fn get_account_info(
        &self,
        pubkey_str: String,
        config: Option<RpcAccountInfoConfig>,
    ) -> RpcResult<RpcResponse<Option<UiAccount>>> {
        // 1. Pubkey 파싱
        let pubkey = Pubkey::from_str(&pubkey_str)
            .map_err(|_| RpcCustomError::InvalidParams)?;

        // 2. 뱅크 가져오기 (최신 또는 특정 슬롯)
        let bank = self.get_bank_with_config(&config)?;

        // 3. 계정 로드
        let account = bank.get_account(&pubkey);

        // 4. UI 형식으로 변환
        let ui_account = account.map(|acc| UiAccount {
            lamports: acc.lamports,
            data: UiAccountData::encode(acc.data, config.encoding),
            owner: acc.owner.to_string(),
            executable: acc.executable,
            rent_epoch: acc.rent_epoch,
        });

        // 5. 응답 (컨텍스트 포함)
        Ok(RpcResponse {
            context: RpcResponseContext {
                slot: bank.slot(),
                api_version: Some(solana_version::semver!()),
            },
            value: ui_account,
        })
    }

    // 트랜잭션 전송
    fn send_transaction(
        &self,
        data: String,
        config: Option<RpcSendTransactionConfig>,
    ) -> RpcResult<String> {
        // 1. Base64/Base58 디코딩
        let tx_bytes = bs58::decode(&data).into_vec()
            .map_err(|_| RpcCustomError::InvalidParams)?;

        // 2. 트랜잭션 역직렬화
        let tx: Transaction = bincode::deserialize(&tx_bytes)
            .map_err(|_| RpcCustomError::InvalidParams)?;

        // 3. 서명 검증 (옵션)
        if config.skip_preflight != Some(true) {
            tx.verify()?;

            // 프리플라이트 시뮬레이션
            let bank = self.bank_forks.read().unwrap().working_bank();
            bank.simulate_transaction(&tx)?;
        }

        // 4. TPU로 전송 (UDP)
        // TPU = Transaction Processing Unit (리더)
        let tpu_address = self.get_current_leader_tpu()?;

        let socket = UdpSocket::bind("0.0.0.0:0")?;
        socket.send_to(&tx_bytes, tpu_address)?;

        // 5. 트랜잭션 해시 반환
        Ok(tx.signatures[0].to_string())
    }

    // 블록 조회
    fn get_block(
        &self,
        slot: Slot,
        config: Option<RpcBlockConfig>,
    ) -> RpcResult<Option<UiConfirmedBlock>> {
        // 1. 블록스토어에서 조회
        let confirmed_block = self.blockstore
            .get_confirmed_block(slot)
            .map_err(|_| RpcCustomError::SlotSkipped)?;

        // 2. UI 형식으로 변환
        let ui_block = UiConfirmedBlock {
            previous_blockhash: confirmed_block.previous_blockhash.to_string(),
            blockhash: confirmed_block.blockhash.to_string(),
            parent_slot: confirmed_block.parent_slot,
            transactions: confirmed_block.transactions
                .into_iter()
                .map(|tx_with_meta| {
                    UiTransactionEncoding::encode(tx_with_meta, config.encoding)
                })
                .collect(),
            rewards: confirmed_block.rewards,
            block_time: confirmed_block.block_time,
            block_height: confirmed_block.block_height,
        };

        Ok(Some(ui_block))
    }

    // 다중 계정 조회
    fn get_multiple_accounts(
        &self,
        pubkeys: Vec<String>,
        config: Option<RpcAccountInfoConfig>,
    ) -> RpcResult<RpcResponse<Vec<Option<UiAccount>>>> {
        // 1. Pubkey 파싱
        let pubkeys: Vec<Pubkey> = pubkeys
            .into_iter()
            .map(|s| Pubkey::from_str(&s))
            .collect::<Result<_, _>>()
            .map_err(|_| RpcCustomError::InvalidParams)?;

        // 2. 뱅크 가져오기
        let bank = self.get_bank_with_config(&config)?;

        // 3. 배치 로드 (최적화!)
        // 의도: 한 번의 디스크 접근으로 여러 계정 로드
        let accounts = bank.get_accounts(&pubkeys);

        // 4. UI 형식으로 변환
        let ui_accounts = accounts
            .into_iter()
            .map(|opt_acc| opt_acc.map(|acc| UiAccount::encode(...)))
            .collect();

        Ok(RpcResponse {
            context: RpcResponseContext { slot: bank.slot(), ... },
            value: ui_accounts,
        })
    }
}
```

### 4.2 PubSub (WebSocket)

**소스 위치**: `rpc/src/rpc_pubsub.rs`

```rust
// PubSub: WebSocket 기반 구독 서비스
// 목적: 실시간 이벤트 알림

use jsonrpc_pubsub::{PubSubHandler, Session, Subscriber, SubscriptionId};

pub trait RpcSolPubSub {
    // 계정 구독
    #[pubsub(subscription = "accountNotification", subscribe, name = "accountSubscribe")]
    fn account_subscribe(
        &self,
        meta: Self::Metadata,
        subscriber: Subscriber<RpcResponse<UiAccount>>,
        pubkey: String,
        config: Option<RpcAccountInfoConfig>,
    );

    // 계정 구독 해제
    #[pubsub(subscription = "accountNotification", unsubscribe, name = "accountUnsubscribe")]
    fn account_unsubscribe(
        &self,
        meta: Option<Self::Metadata>,
        id: SubscriptionId,
    ) -> RpcResult<bool>;

    // 슬롯 구독
    #[pubsub(subscription = "slotNotification", subscribe, name = "slotSubscribe")]
    fn slot_subscribe(&self, meta: Self::Metadata, subscriber: Subscriber<SlotInfo>);

    // 로그 구독 (트랜잭션 로그)
    #[pubsub(subscription = "logsNotification", subscribe, name = "logsSubscribe")]
    fn logs_subscribe(
        &self,
        meta: Self::Metadata,
        subscriber: Subscriber<RpcResponse<RpcLogsResponse>>,
        filter: RpcTransactionLogsFilter,
        config: Option<RpcTransactionLogsConfig>,
    );

    // 시그니처 구독 (트랜잭션 확인)
    #[pubsub(subscription = "signatureNotification", subscribe, name = "signatureSubscribe")]
    fn signature_subscribe(
        &self,
        meta: Self::Metadata,
        subscriber: Subscriber<RpcResponse<RpcSignatureResult>>,
        signature: String,
        config: Option<RpcSignatureSubscribeConfig>,
    );
}

// 구독 관리자
pub struct RpcSubscriptions {
    // 계정 구독자들: Pubkey -> Vec<Subscriber>
    account_subscriptions: RwLock<HashMap<Pubkey, Vec<Subscription>>>,

    // 슬롯 구독자들
    slot_subscriptions: RwLock<Vec<Subscription>>,

    // 시그니처 구독자들: Signature -> Subscriber
    signature_subscriptions: RwLock<HashMap<Signature, Subscription>>,

    // 이벤트 수신 채널
    notification_sender: Arc<RpcNotificationSender>,
}

impl RpcSubscriptions {
    // 계정 변경 알림
    pub fn notify_account_update(
        &self,
        slot: Slot,
        pubkey: &Pubkey,
        account: &Account,
    ) {
        // 1. 해당 계정을 구독하는 클라이언트들 찾기
        let subs = self.account_subscriptions.read().unwrap();
        if let Some(subscribers) = subs.get(pubkey) {
            // 2. UI 형식으로 변환
            let ui_account = UiAccount::encode(account, ...);

            // 3. 각 구독자에게 알림
            for sub in subscribers {
                let notification = RpcResponse {
                    context: RpcResponseContext { slot, ... },
                    value: ui_account.clone(),
                };

                // WebSocket으로 전송
                sub.sink.notify(Ok(notification));
            }
        }
    }

    // 슬롯 변경 알림
    pub fn notify_slot(&self, slot: Slot, parent: Slot, root: Slot) {
        let subs = self.slot_subscriptions.read().unwrap();

        let slot_info = SlotInfo {
            slot,
            parent,
            root,
        };

        for sub in subs.iter() {
            sub.sink.notify(Ok(slot_info));
        }
    }

    // 트랜잭션 확인 알림
    pub fn notify_signature(
        &self,
        signature: &Signature,
        result: &TransactionResult,
    ) {
        let mut subs = self.signature_subscriptions.write().unwrap();

        if let Some(sub) = subs.remove(signature) {
            // 한 번만 알림하고 제거
            let response = RpcResponse {
                context: ...,
                value: RpcSignatureResult { err: result.err },
            };

            sub.sink.notify(Ok(response));
        }
    }
}

// 통합 (Bank 커밋시 호출)
impl Bank {
    pub fn commit_transactions(&self, txs: &[Transaction]) -> Vec<TransactionResult> {
        // ... 트랜잭션 실행 ...

        // 구독자들에게 알림
        for (pubkey, account) in &changed_accounts {
            self.subscriptions.notify_account_update(self.slot(), pubkey, account);
        }

        for (signature, result) in &tx_results {
            self.subscriptions.notify_signature(signature, result);
        }

        // ...
    }
}
```

---

## 5. 핵심 아키텍처 컴포넌트

### 5.1 Proof of History (PoH)

**소스 위치**: `poh/src/poh_recorder.rs`

```rust
// PoH: 시간의 암호학적 증명
// 목적: 검증 가능한 시간 순서 제공

pub struct PohRecorder {
    // PoH 해시 체인
    poh: Arc<Mutex<Poh>>,

    // 틱 생성 간격
    tick_duration: Duration,

    // 현재 작업 뱅크
    working_bank: Option<Arc<Bank>>,
}

pub struct Poh {
    // 현재 해시
    hash: Hash,

    // 틱 카운터
    num_hashes: u64,

    // 해시 레이트 (hashes/tick)
    hashes_per_tick: u64,
}

impl Poh {
    // PoH 해시 체인 진행
    pub fn hash(&mut self) {
        // SHA-256 해시 체인
        // hash_n = SHA256(hash_{n-1})
        self.hash = solana_sdk::hash::hashv(&[self.hash.as_ref()]);
        self.num_hashes += 1;
    }

    // 틱 생성
    pub fn tick(&mut self) -> Option<PohEntry> {
        // 목적: 시간 단위 (기본 6.25ms)

        // 1. 정해진 횟수만큼 해시
        for _ in 0..self.hashes_per_tick {
            self.hash();
        }

        // 2. 틱 엔트리 생성
        let tick_entry = PohEntry {
            num_hashes: self.num_hashes,
            hash: self.hash,
            transactions: vec![],  // 틱은 트랜잭션 없음
        };

        self.num_hashes = 0;
        Some(tick_entry)
    }

    // 트랜잭션 믹싱
    pub fn record(&mut self, mixin: Hash) -> PohEntry {
        // 목적: 트랜잭션을 PoH 스트림에 삽입

        // 1. 트랜잭션 해시를 PoH에 믹스
        // hash = SHA256(prev_hash || tx_hash)
        self.hash = solana_sdk::hash::hashv(&[
            self.hash.as_ref(),
            mixin.as_ref(),
        ]);

        self.num_hashes += 1;

        // 2. 엔트리 생성
        PohEntry {
            num_hashes: self.num_hashes,
            hash: self.hash,
            transactions: vec![/* 트랜잭션들 */],
        }
    }
}

// PoH 검증
pub fn verify_poh_entries(entries: &[Entry]) -> bool {
    // 목적: PoH 체인의 무결성 검증

    let mut current_hash = Hash::default();

    for entry in entries {
        // 1. 해시 재계산
        let mut hash = current_hash;

        // 트랜잭션 믹스
        for tx in &entry.transactions {
            hash = solana_sdk::hash::hashv(&[hash.as_ref(), tx.hash().as_ref()]);
        }

        // 나머지 해시들
        for _ in 0..entry.num_hashes {
            hash = solana_sdk::hash::hashv(&[hash.as_ref()]);
        }

        // 2. 해시 일치 확인
        if hash != entry.hash {
            return false;  // 위조 감지!
        }

        current_hash = hash;
    }

    true  // 검증 성공
}

// 특징:
// - 병렬 검증 불가능 (순차적)
// - 시간 경과 증명
// - 리더 스케줄링의 기반
```

### 5.2 Tower BFT (Consensus)

**소스 위치**: `core/src/consensus/tower.rs`

```rust
// Tower BFT: Solana의 합의 알고리즘
// 목적: PoH 기반 BFT 합의

pub struct Tower {
    // 투표 타워 (스택)
    // 목적: 투표 이력 추적
    votes: VecDeque<Vote>,

    // 루트 (finalized)
    root: Slot,

    // 타임아웃
    last_vote_time: Instant,
}

pub struct Vote {
    slot: Slot,              // 투표한 슬롯
    confirmation_count: u32, // 확인 횟수 (타워 높이)
}

impl Tower {
    // 투표 결정
    pub fn check_vote_stake_threshold(
        &self,
        slot: Slot,
        stake_lockouts: &HashMap<Slot, StakeLockout>,
        total_stake: u64,
    ) -> bool {
        // 목적: 2/3 스테이크 확인

        let stake = stake_lockouts.get(&slot)
            .map(|s| s.stake)
            .unwrap_or(0);

        // 슈퍼 majority (66.67%)
        stake * 3 > total_stake * 2
    }

    // 투표 추가
    pub fn record_vote(&mut self, slot: Slot) {
        // 1. 새 투표 추가
        self.votes.push_back(Vote {
            slot,
            confirmation_count: 1,
        });

        // 2. 타워 업데이트
        // 이전 투표들의 confirmation_count 증가
        for vote in self.votes.iter_mut().rev().skip(1) {
            if vote.slot < slot {
                vote.confirmation_count += 1;
            }
        }

        // 3. Lockout 계산
        // lockout_distance = 2^confirmation_count
        // 예: confirmation=5 -> lockout=32 슬롯
    }

    // Finality 확인
    pub fn check_finality(&mut self) -> Option<Slot> {
        // 목적: 슬롯이 finalized 되었는지 확인

        // 슬래싱 위험 없이 롤백 불가능한 시점
        // 조건: confirmation_count >= MAX_LOCKOUT (32)

        for vote in &self.votes {
            if vote.confirmation_count >= MAX_LOCKOUT {
                // Finalized!
                self.root = vote.slot;
                return Some(vote.slot);
            }
        }

        None
    }
}

// 리더 스케줄
pub fn calculate_leader_schedule(
    epoch: Epoch,
    stakes: &HashMap<Pubkey, u64>,
) -> Vec<Pubkey> {
    // 목적: 에포크의 슬롯별 리더 결정

    const SLOTS_PER_EPOCH: u64 = 432_000;  // ~2일

    // 1. 총 스테이크
    let total_stake: u64 = stakes.values().sum();

    // 2. 각 검증자의 슬롯 수 (스테이크 비례)
    let mut leader_schedule = Vec::with_capacity(SLOTS_PER_EPOCH as usize);

    for (pubkey, stake) in stakes {
        let num_slots = (SLOTS_PER_EPOCH * stake) / total_stake;

        for _ in 0..num_slots {
            leader_schedule.push(*pubkey);
        }
    }

    // 3. 셔플 (deterministic, epoch seed 사용)
    let seed = Hash::new(&epoch.to_le_bytes());
    shuffle(&mut leader_schedule, seed);

    leader_schedule
}
```

### 5.3 Banking Stage (트랜잭션 실행)

**소스 위치**: `core/src/banking_stage.rs`

```rust
// BankingStage: 트랜잭션 배치 실행
// 목적: 병렬 트랜잭션 처리

pub struct BankingStage {
    // 입력: 트랜잭션 배치
    packet_receiver: Receiver<PacketBatch>,

    // 출력: PoH 레코더
    poh_recorder: Arc<Mutex<PohRecorder>>,

    // 실행 스레드들
    num_threads: usize,
}

impl BankingStage {
    pub fn process_packets(&self, bank: &Bank, packets: &PacketBatch) {
        // === 1단계: 서명 검증 (GPU) ===
        // 목적: CPU 부하 감소
        let verified = verify_signatures_gpu(packets);

        // === 2단계: 계정 충돌 감지 ===
        // 목적: 병렬 실행 가능한 트랜잭션 그룹화

        let account_locks = self.lock_accounts(&verified, bank);

        // 충돌 그래프 생성
        // tx1: [A, B] 사용
        // tx2: [C, D] 사용  -> 병렬 가능!
        // tx3: [A, E] 사용  -> tx1과 충돌

        let batches = self.create_non_conflicting_batches(account_locks);

        // === 3단계: 병렬 실행 ===
        // 각 배치를 별도 스레드에서 실행

        let results: Vec<_> = batches
            .par_iter()  // Rayon parallel iterator
            .map(|batch| {
                // 배치 내 트랜잭션들은 순차 실행 (충돌 있음)
                // 배치 간은 병렬 실행 (충돌 없음!)
                self.execute_batch(bank, batch)
            })
            .collect();

        // === 4단계: 결과 커밋 ===
        bank.commit_transactions(&results);

        // === 5단계: PoH에 기록 ===
        let poh = self.poh_recorder.lock().unwrap();
        for tx in &verified {
            poh.record(tx.hash());
        }
    }

    fn lock_accounts(
        &self,
        txs: &[Transaction],
        bank: &Bank,
    ) -> Vec<AccountLocks> {
        // 목적: 트랜잭션이 접근하는 계정 식별

        txs.iter()
            .map(|tx| {
                let mut locks = AccountLocks::default();

                // 읽기 전용 계정들
                for key in &tx.message.account_keys[..tx.message.num_readonly_accounts] {
                    locks.readonly.insert(*key);
                }

                // 쓰기 가능 계정들
                for key in &tx.message.account_keys[tx.message.num_readonly_accounts..] {
                    locks.writable.insert(*key);
                }

                locks
            })
            .collect()
    }

    fn create_non_conflicting_batches(
        &self,
        locks: Vec<AccountLocks>,
    ) -> Vec<Vec<usize>> {  // 트랜잭션 인덱스들
        // 목적: 충돌 없는 트랜잭션들을 같은 배치로

        let mut batches = Vec::new();
        let mut used_accounts = HashSet::new();

        let mut current_batch = Vec::new();

        for (i, lock) in locks.iter().enumerate() {
            // 쓰기 충돌 확인
            let conflicts = lock.writable.iter()
                .any(|key| used_accounts.contains(key));

            if conflicts {
                // 새 배치 시작
                batches.push(current_batch);
                current_batch = Vec::new();
                used_accounts.clear();
            }

            current_batch.push(i);
            used_accounts.extend(lock.writable.iter());
        }

        if !current_batch.is_empty() {
            batches.push(current_batch);
        }

        batches
    }
}
```

---

## 학습 로드맵

### Phase 1: 기초 이해 (1-2주)
1. **Gossip 프로토콜**
   - `gossip/src/cluster_info.rs` 읽기
   - CRDS 데이터 구조 이해
   - 로컬 테스트넷 구성

2. **PoH 개념**
   - `poh/src/poh_recorder.rs` 분석
   - PoH 엔트리 생성 및 검증 실습
   - 시간 경과 증명 이해

### Phase 2: 심화 학습 (2-3주)
1. **Turbine & Shreds**
   - `ledger/src/shred.rs` 구조 분석
   - Reed-Solomon 코딩 이해
   - Turbine tree 시뮬레이션

2. **AccountsDB**
   - `runtime/src/accounts_db.rs` 읽기
   - AppendVec 구조 이해
   - 스냅샷 생성/로드 실습

### Phase 3: 전문가 (3-4주)
1. **Banking Stage**
   - 병렬 실행 메커니즘 이해
   - 계정 충돌 감지 알고리즘
   - 성능 최적화 기법

2. **Tower BFT**
   - 합의 알고리즘 상세 분석
   - 리더 스케줄링
   - Finality 메커니즘

### 실습 프로젝트
1. Gossip 네트워크 모니터
2. PoH 검증기
3. 계정 변경 추적 도구
4. 트랜잭션 시뮬레이터

---

## 추가 학습 자료

### 공식 문서
- Solana 문서: https://docs.solana.com
- Solana Cookbook: https://solanacookbook.com
- Agave 문서: https://docs.anza.xyz

### 소스 코드
- Agave 저장소: https://github.com/anza-xyz/agave (구 solana-labs/solana)
- 주요 디렉토리:
  - `gossip/`: Gossip 프로토콜
  - `poh/`: Proof of History
  - `ledger/`: Blockstore, Shreds
  - `runtime/`: AccountsDB, Bank
  - `core/`: 합의, 실행 파이프라인
  - `rpc/`: JSON-RPC

### 성능 분석
```bash
# 검증자 실행 (테스트넷)
solana-test-validator

# RPC 호출
solana balance <address>
solana block <slot>

# 성능 모니터링
solana-watchtower
```

---

## 6. 완전한 트랜잭션 처리 플로우 (Solana)

### 6.1 End-to-End Transaction Path

**시나리오**: 사용자가 Token Transfer 트랜잭션 전송 (100 SOL)

#### Step 1: RPC 요청 수신
**파일**: `rpc/src/rpc.rs:412`

```rust
// sendTransaction 처리 시작
pub fn send_transaction(&self, data: String, config: RpcSendTransactionConfig) -> Result<String> {
    // 1-1. Base64 디코딩
    let tx_data = bs58::decode(&data).into_vec()?;
    let mut tx: VersionedTransaction = bincode::deserialize(&tx_data)?;
    
    // 1-2. 트랜잭션 검증
    let signature = tx.signatures[0];
    
    // Recent blockhash 확인
    let recent_blockhash = self.bank().last_blockhash();
    if tx.message.recent_blockhash() != recent_blockhash {
        return Err(RpcCustomError::BlockhashNotFound);
    }
    
    // 1-3. Signature 검증 (여러 서명 가능)
    tx.verify_and_hash_message()?;
    
    // 1-4. TPU로 전송
    self.send_transaction_service.send(tx)?;
    
    Ok(signature.to_string())
}
```

**검증 항목**:
- Message size < 1232 bytes
- Signatures 개수 <= 12
- Account keys 개수 <= 128
- Recent blockhash 유효 (150 slots 이내)

#### Step 2: TPU (Transaction Processing Unit)
**파일**: `core/src/banking_stage/mod.rs:286`

```rust
impl BankingStage {
    fn process_buffered_packets(&self, bank: &Bank) -> BufferedPacketsDecision {
        // 2-1. 패킷 배치 생성 (64개씩)
        let packets = self.receive_and_buffer_packets();
        
        // 2-2. 서명 검증 (GPU 가속)
        let verified_packets = self.verify_signatures(packets)?;
        
        // 2-3. 계정 Lock 분석
        //     동일 계정 접근 트랜잭션 → Sequential
        //     독립적인 계정 → Parallel
        let batches = self.prepare_batches(verified_packets);
        
        // 2-4. Banking Stage 실행
        for batch in batches {
            self.process_batch(bank, batch)?;
        }
        
        Ok(BufferedPacketsDecision::Consume)
    }
}
```

**Signature 검증** (`core/src/sigverify_stage.rs:127`):

```rust
// GPU를 사용한 병렬 서명 검증
fn verify_batch_signatures(batch: &[Packet]) -> Vec<bool> {
    let mut results = vec![false; batch.len()];
    
    // CPU vs GPU 선택
    if batch.len() > 128 && has_cuda_device() {
        // GPU (CUDA) - 수천 개 병렬
        gpu::verify_signatures_cuda(batch, &mut results);
    } else {
        // CPU (multi-thread) - 수십 개 병렬
        batch.par_iter().enumerate().for_each(|(i, pkt)| {
            results[i] = pkt.meta.signature.verify(&pkt.meta.pubkey, &pkt.data);
        });
    }
    
    results
}
```

**성능**:
- CPU: ~10,000 signatures/sec
- GPU: ~50,000+ signatures/sec

#### Step 3: Account Locking & Scheduling
**파일**: `runtime/src/bank.rs:3891`

```rust
fn load_execute_and_commit_transactions(&self, batch: &TransactionBatch) -> TransactionResults {
    // 3-1. Account locks 획득
    //     Read lock: 읽기만 하는 계정
    //     Write lock: 쓰기 하는 계정
    let lock_results = self.lock_accounts(batch.transactions());
    
    // Lock 충돌 감지:
    // Tx1: Transfer A → B (locks: A write, B write)
    // Tx2: Transfer A → C (locks: A write, C write)
    // → Conflict! Sequential 실행 필요
    
    // 3-2. 병렬 실행 배치 구성
    let batches = self.prepare_parallel_batches(&lock_results);
    
    // 3-3. 병렬 실행
    let results = self.execute_batches_in_parallel(batches);
    
    // 3-4. Locks 해제
    self.unlock_accounts(batch);
    
    results
}
```

**병렬 실행 전략**:
```rust
// Rayon을 사용한 병렬 처리
results.par_iter_mut().enumerate().for_each(|(i, result)| {
    // 각 스레드가 독립적인 트랜잭션 실행
    *result = execute_transaction(&bank, &txs[i]);
});
```

#### Step 4: 프로그램 실행 (Runtime)
**파일**: `program-runtime/src/invoke_context.rs:612`

```rust
fn process_instruction(&mut self, instruction_data: &[u8]) -> Result<()> {
    // 4-1. 프로그램 로드
    let program_id = self.transaction_context.get_instruction_program_id()?;
    let program_account = self.get_account(program_id)?;
    
    // 4-2. BPF VM 초기화
    let mut vm = create_vm(
        &program_account.data,
        &self.accounts,
        &self.invoke_stack,
    )?;
    
    // 4-3. 실행 (Compute Units 제한)
    let compute_meter = self.compute_meter;
    compute_meter.consume(DEFAULT_COMPUTE_UNITS)?;
    
    let result = vm.execute_program_jit(
        instruction_data,
        &mut self.accounts,
        &self.instruction_data,
    )?;
    
    // 4-4. Compute Units 소진 확인
    if compute_meter.get_remaining() == 0 {
        return Err(InstructionError::ComputationalBudgetExceeded);
    }
    
    Ok(result)
}
```

**Compute Units**:
```
기본 제한: 200,000 CU
추가 요청: Max 1,400,000 CU (추가 fee)

비용:
- SYSVAR read: 100 CU
- Account 생성: 23,000 CU
- SHA256: 20 CU/byte
- Ed25519 verify: 3,000 CU
- Transfer: 300 CU
```

#### Step 5: AccountsDB 업데이트
**파일**: `runtime/src/accounts_db.rs:2518`

```rust
fn store_cached(&self, slot: Slot, accounts: &[(&Pubkey, &Account)]) {
    // 5-1. AppendVec 선택/생성
    let storage = self.find_storage_candidate(slot, accounts.len())?;
    
    // 5-2. 순차 쓰기 (Append-only)
    let mut offsets = Vec::with_capacity(accounts.len());
    
    for (pubkey, account) in accounts {
        // 직렬화
        let serialized = serialize_account(account);
        
        // AppendVec에 추가
        let offset = storage.append_account(serialized)?;
        offsets.push((*pubkey, offset));
        
        // 5-3. AccountsIndex 업데이트 (메모리)
        self.accounts_index.upsert(
            slot,
            *pubkey,
            &AccountInfo {
                store_id: storage.id(),
                offset,
                lamports: account.lamports,
            },
        );
    }
    
    // 5-4. Storage Map 업데이트
    self.storage.insert(storage.id(), storage);
}
```

**AppendVec 구조**:
```
파일: accounts.X
├─ Account 1 (offset 0)
├─ Account 2 (offset 512)
├─ Account 3 (offset 1024)
└─ ...

AccountsIndex (메모리):
Pubkey1 → (store_id=X, offset=0)
Pubkey2 → (store_id=X, offset=512)
```

#### Step 6: PoH (Proof of History) 기록
**파일**: `poh/src/poh_recorder.rs:178`

```rust
fn record_transaction(&mut self, hash: Hash) -> Result<()> {
    // 6-1. 현재 PoH hash에 tx hash mix
    self.poh.record(hash)?;
    
    // PoH record 구조:
    // prev_hash = SHA256(prev_hash)  <- tick
    // curr_hash = SHA256(prev_hash || tx_hash)  <- transaction
    
    // 6-2. Entry 생성
    let num_hashes = self.poh.tick_height - self.last_entry_tick;
    
    let entry = Entry {
        num_hashes,
        hash: self.poh.hash,
        transactions: vec![hash],
    };
    
    // 6-3. Shred 생성 (나중에 broadcast)
    self.working_bank.add_entry(entry);
    
    Ok(())
}
```

#### Step 7: Shred 생성 및 Broadcast
**파일**: `ledger/src/shred.rs:523`

```rust
fn make_shreds_from_entries(entries: &[Entry], slot: Slot, leader: &Pubkey) -> Vec<Shred> {
    // 7-1. Entry를 바이트로 직렬화
    let serialized = bincode::serialize(entries)?;
    
    // 7-2. ~1KB 청크로 분할
    const MAX_DATA_SHREDS_PER_FEC_BLOCK: usize = 67;
    let chunks: Vec<_> = serialized.chunks(MAX_SHRED_PAYLOAD_SIZE).collect();
    
    let mut data_shreds = Vec::new();
    for (i, chunk) in chunks.iter().enumerate() {
        // Data Shred 생성
        let shred = Shred::new_from_data(
            slot,
            i as u32,  // index
            0,         // parent offset
            chunk,
            true,      // is_last_in_slot
            i as u8,   // fec_set_index
        );
        data_shreds.push(shred);
    }
    
    // 7-3. Reed-Solomon FEC 코딩
    //     67 Data Shreds → 33 Coding Shreds
    let coding_shreds = generate_coding_shreds(&data_shreds, 67, 33)?;
    
    // 7-4. Signature 추가
    for shred in data_shreds.iter_mut().chain(coding_shreds.iter_mut()) {
        shred.sign(leader);
    }
    
    data_shreds.extend(coding_shreds);
    data_shreds
}
```

**Turbine Broadcast** (`core/src/broadcast_stage/broadcast_shreds.rs:142`):

```rust
fn broadcast(&self, shreds: Vec<Shred>) -> Result<()> {
    // Turbine tree 구조
    // Layer 0 (Leader): 1 노드
    // Layer 1: 200 노드 (DATA_PLANE_FANOUT)
    // Layer 2: 200*200 = 40,000 노드
    
    let peers = self.get_broadcast_peers();
    
    // Stake-weighted 정렬 (높은 stake 먼저)
    let sorted_peers = self.sort_peers_by_stake(peers);
    
    // 각 shred를 여러 피어에게 전송
    for (shred_index, shred) in shreds.iter().enumerate() {
        let target_peers = &sorted_peers[shred_index % sorted_peers.len()..];
        
        for peer in target_peers.iter().take(DATA_PLANE_FANOUT) {
            self.send_shred_to_peer(peer, shred)?;
        }
    }
    
    Ok(())
}
```

### 6.2 실제 성능 분석

**Mainnet Metrics** (2024년 기준):

```
TPS (transactions per second): 3,000-5,000
Slot time: 400ms
Transactions per slot: 1,200-2,000
Block propagation: < 200ms (95 percentile)

Stage별 시간:
────────────────────────────────
SigVerify (GPU):        ~10ms
Banking Stage:          ~50ms
  ├─ Lock accounts:     5ms
  ├─ Execute (parallel):35ms
  └─ Commit:            10ms
PoH Record:             ~5ms
Shred generation:       ~15ms
Turbine broadcast:      ~120ms
Total:                  ~200ms
```

**병목 지점**:
1. **Account contention** (35%): 인기 계정(DEX pools)에 대한 lock 경쟁
2. **Compute Units** (25%): 복잡한 프로그램 실행
3. **Network bandwidth** (20%): Shred 전파
4. **SigVerify** (10%): CPU/GPU 처리량
5. **기타** (10%)

---

## 7. 성능 최적화 실전 (Solana)

### 7.1 Parallel Execution 최적화

**문제**: Account contention으로 병렬성 저하

**해결**: Account Prefetching

```rust
// runtime/src/bank.rs:4102
fn prefetch_accounts(&self, txs: &[Transaction]) {
    // 트랜잭션 실행 전에 계정 미리 로드
    let accounts_to_load: HashSet<Pubkey> = txs
        .iter()
        .flat_map(|tx| tx.message.account_keys.iter())
        .cloned()
        .collect();
    
    // 병렬로 AccountsDB에서 로드
    accounts_to_load.par_iter().for_each(|pubkey| {
        self.accounts_db.load_account(pubkey);  // Cache에 저장
    });
}
```

**효과**:
- Cache hit rate: 70% → 95%
- Execute time: 50ms → 35ms (-30%)

### 7.2 AccountsDB Compaction

**문제**: AppendVec 파일이 계속 증가 (디스크 낭비)

**Compaction 전략**:

```rust
// runtime/src/accounts_db.rs:3241
fn shrink_candidate_slots(&self) {
    // 7.1. 낮은 utilization AppendVec 찾기
    for storage in self.storage.values() {
        let alive_bytes = self.calc_alive_bytes(storage);
        let total_bytes = storage.capacity();
        
        let utilization = (alive_bytes as f64) / (total_bytes as f64);
        
        if utilization < 0.80 {
            // 80% 미만 → Compaction 대상
            self.shrink_storage(storage)?;
        }
    }
}

fn shrink_storage(&self, old_storage: &AccountStorage) -> Result<()> {
    // 7.2. 살아있는 계정만 새 AppendVec에 복사
    let new_storage = self.create_storage();
    
    for (pubkey, account_info) in old_storage.accounts() {
        if self.is_account_alive(pubkey, account_info) {
            let account = old_storage.get_account(account_info.offset);
            new_storage.append_account(account);
        }
    }
    
    // 7.3. AccountsIndex 업데이트
    self.accounts_index.update_storage(old_storage.id(), new_storage.id());
    
    // 7.4. 이전 파일 삭제
    std::fs::remove_file(old_storage.path())?;
    
    Ok(())
}
```

**Compaction 시점**:
- Background thread (low priority)
- Utilization < 80%
- Disk space > 90% full (aggressive)

### 7.3 PoH Hash Acceleration

**문제**: PoH는 sequential bottleneck

**최적화**: AVX2/AVX512 SIMD

```rust
// poh/src/poh_service.rs:87
fn hash_with_simd(prev_hash: &Hash, data: &[u8]) -> Hash {
    #[cfg(target_arch = "x86_64")]
    {
        if is_x86_feature_detected!("avx512f") {
            return sha256_avx512(prev_hash, data);
        } else if is_x86_feature_detected!("avx2") {
            return sha256_avx2(prev_hash, data);
        }
    }
    
    // Fallback
    sha256_generic(prev_hash, data)
}
```

**성능 비교**:
```
Generic:   800,000 hashes/sec
AVX2:    1,200,000 hashes/sec (+50%)
AVX512:  1,600,000 hashes/sec (+100%)
```

### 7.4 Shred FEC 최적화

**Reed-Solomon 인코딩** (`ledger/src/erasure.rs:214`):

```rust
// GPU 가속 FEC 인코딩
fn generate_coding_shreds_gpu(data_shreds: &[Shred], num_data: usize, num_coding: usize) -> Vec<Shred> {
    // CPU (single-threaded): ~50ms/FEC block
    // GPU (CUDA): ~5ms/FEC block (10x faster)
    
    if has_cuda_device() && data_shreds.len() > 32 {
        cuda_rs_encode(data_shreds, num_data, num_coding)
    } else {
        cpu_rs_encode(data_shreds, num_data, num_coding)
    }
}
```

---

## 8. 디버깅 및 트러블슈팅 (Solana)

### 8.1 RPC 디버깅 도구

#### 8.1.1 Transaction Simulation

```bash
# 트랜잭션 실행 전 시뮬레이션
curl https://api.mainnet-beta.solana.com -X POST -H "Content-Type: application/json" -d '
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "simulateTransaction",
  "params": [
    "<base64_transaction>",
    {"commitment": "processed"}
  ]
}'

# Response:
{
  "result": {
    "err": null,  # 성공
    "logs": [
      "Program 11111111111111111111111111111111 invoke [1]",
      "Program 11111111111111111111111111111111 success"
    ],
    "unitsConsumed": 150
  }
}
```

#### 8.1.2 Program Logs

```bash
# getProgramAccounts로 계정 조회
solana account <PUBKEY> --output json-compact

# 트랜잭션 상세 조회
solana confirm -v <SIGNATURE>

# Output:
Transaction executed in slot 123456789:
  Block Time: 2024-01-15T10:30:00Z
  Recent Blockhash: ABC123...
  Signature: DEF456...
  Account 0: signer, writable, 0.001 SOL
  Account 1: writable, 0 SOL
  Instruction 0: Transfer 100000000 lamports
    Program: 11111111111111111111111111111111
    Logs:
      - "Transfer: 0.1 SOL"
  Status: Ok
```

### 8.2 일반적인 에러 및 해결

#### 에러 1: "Blockhash Not Found"

**원인**: Recent blockhash 만료 (150 slots ≈ 60초)

```rust
// 해결: 최신 blockhash 조회 및 재전송
let recent_blockhash = rpc_client.get_latest_blockhash()?;
transaction.message.recent_blockhash = recent_blockhash;
transaction.sign(&[&payer], recent_blockhash);
rpc_client.send_transaction(&transaction)?;
```

#### 에러 2: "Insufficient Funds for Fee"

**원인**: 트랜잭션 fee를 낼 수 없음

```rust
// Fee 계산:
// fee = signatures × lamports_per_signature
// lamports_per_signature = 5000 (현재)

// 예: 2 signatures → 10,000 lamports (0.00001 SOL)

// 해결: 최소 잔액 확인
let balance = rpc_client.get_balance(&payer.pubkey())?;
let required = rent_exempt_minimum + fee;

if balance < required {
    println!("Need at least {} lamports", required);
}
```

#### 에러 3: "ComputationalBudgetExceeded"

**원인**: Compute Units 초과 (기본 200K CU)

```rust
// 해결: Compute Budget 증가 요청
let increase_budget_ix = ComputeBudgetInstruction::set_compute_unit_limit(400_000);

let mut transaction = Transaction::new_with_payer(
    &[
        increase_budget_ix,  // 첫 번째 instruction
        main_instruction,
    ],
    Some(&payer.pubkey()),
);
```

**추가 fee**: 
```
CU 증가량 당 fee:
200K → 400K CU: +0.000005 SOL
200K → 1.4M CU (max): +0.00003 SOL
```

### 8.3 Validator 로그 분석

```bash
# 로그 레벨 설정
solana-validator --log - --rpc-port 8899 \
  --log-level info \
  --log-messages-bytes-limit 1000000

# 주요 로그 패턴:
[INFO] Slot 12345 completed in 423ms
[INFO] Processed 1542 transactions, 23 failed
[WARN] Skipped 5 slots due to network issues
[ERROR] Bank fork rejected: InvalidBlockhash

# 로그 필터링
tail -f validator.log | grep -E "ERROR|WARN"
```

---

## 9. 프로덕션 환경 Best Practices (Solana)

### 9.1 Validator 설정

**하드웨어**:
```
CPU: 12+ cores (AMD EPYC/Intel Xeon)
RAM: 256GB+ (512GB 권장)
Disk: 2TB+ NVMe SSD (PCIe 4.0)
  - Accounts: 500GB
  - Ledger: 500GB
  - Snapshots: 200GB
Network: 1Gbps 대역폭
GPU: NVIDIA RTX 3090 (SigVerify 가속)
```

**설정 파일**:
```bash
#!/bin/bash
# start-validator.sh

solana-validator \
  --identity ~/validator-keypair.json \
  --vote-account ~/vote-account-keypair.json \
  --ledger ~/ledger \
  --accounts ~/accounts \
  --log ~/solana-validator.log \
  --rpc-port 8899 \
  --rpc-bind-address 0.0.0.0 \
  --dynamic-port-range 8000-8020 \
  --entrypoint entrypoint.mainnet-beta.solana.com:8001 \
  --entrypoint entrypoint2.mainnet-beta.solana.com:8001 \
  --known-validator 7Np41oeYqPefeNQEHSv1UDhYrehxin3NStELsSKCT4K2 \
  --known-validator GdnSyH3YtwcxFvQrVVJMm1JhTS4QVX7MFsX56uJLUfiZ \
  --expected-genesis-hash 5eykt4UsFv8P8NJdTREpY1vzqKqZKvdpKuc147dw2N9d \
  --wal-recovery-mode skip_any_corrupted_record \
  --limit-ledger-size 50000000 \
  --block-production-method central-scheduler \
  --full-rpc-api \
  --no-voting \
  --private-rpc
```

### 9.2 모니터링

**Prometheus Metrics**:
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'solana'
    static_configs:
      - targets: ['localhost:8899']
    metrics_path: '/metrics'
```

**주요 메트릭**:
```
# Validator 동기화 상태
solana_validator_health
solana_validator_slot_height
solana_validator_root

# 성능
solana_banking_stage_transactions_processed
solana_replay_stage_time_ms
solana_shred_fetch_stage_packets_received

# 네트워크
solana_cluster_version
solana_validator_delinquent
```

### 9.3 스냅샷 관리

```bash
# 스냅샷 생성 (주기적)
solana-validator \
  --snapshot-interval-slots 500 \
  --maximum-snapshots-to-retain 5

# 스냅샷에서 복구
solana-validator \
  --snapshot ~/snapshots/snapshot-123456789-<hash>.tar.zst \
  --no-genesis-fetch
```

---

## 10. 알려진 이슈 및 해결책 (Solana)

### 10.1 "Slot Skipping" 과다

**현상**: Validator가 슬롯을 자주 skip

**원인**:
- Network latency
- CPU bottleneck
- Disk I/O 느림

**진단**:
```bash
# Validator 통계
solana validators --output json | jq '.validators[] | select(.identityPubkey=="YOUR_PUBKEY")'

# Output:
{
  "skipRate": 15.5,  # 15.5% 슬롯 skip (높음!)
  "lastVote": 123456789,
  "rootSlot": 123456700
}
```

**해결**:
```bash
# 1. 네트워크 최적화
sudo sysctl -w net.core.rmem_max=134217728
sudo sysctl -w net.core.wmem_max=134217728

# 2. CPU governor 설정
sudo cpupower frequency-set -g performance

# 3. Disk I/O 스케줄러
echo "none" | sudo tee /sys/block/nvme0n1/queue/scheduler
```

### 10.2 "AccountsDB Hash Mismatch"

**현상**: Validator가 중단되며 hash mismatch 에러

**원인**: Disk corruption 또는 bug

**복구**:
```bash
# 1. 최신 스냅샷에서 재시작
rm -rf ~/accounts/*
rm -rf ~/ledger/*

solana-validator \
  --no-genesis-fetch \
  --no-snapshot-fetch \
  --snapshot ~/snapshots/latest.tar.zst

# 2. AccountsDB 재구축
solana-ledger-tool verify --ledger ~/ledger
```

### 10.3 OOM (Out of Memory)

**현상**: Validator가 메모리 부족으로 kill됨

**해결**:
```bash
# 1. Swap 설정 (emergency)
sudo fallocate -l 64G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# 2. AccountsDB shrink 주기 감소
solana-validator \
  --accounts-shrink-optimize-total-space \
  --accounts-shrink-ratio 0.8

# 3. 메모리 사용량 모니터링
watch -n 1 'free -h && ps aux | grep solana-validator | grep -v grep'
```

