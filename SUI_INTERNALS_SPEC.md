# Sui 내부 구현 완전 분석

## 목차
1. [네트워크 레이어 (Network Layer)](#1-네트워크-레이어)
2. [데이터베이스 레이어 (Database Layer)](#2-데이터베이스-레이어)
3. [스토리지 레이어 (Storage Layer)](#3-스토리지-레이어)
4. [RPC 레이어 (RPC Layer)](#4-rpc-레이어)
5. [핵심 아키텍처 컴포넌트](#5-핵심-아키텍처-컴포넌트)

---

## 1. 네트워크 레이어 (Network Layer)

### 1.1 Narwhal & Tusk (Consensus Mempool)

**소스 위치**: `narwhal/` (별도 저장소: MystenLabs/narwhal)

#### 1.1.1 핵심 개념
```rust
// Narwhal: DAG 기반 mempool 프로토콜
// 목적: 트랜잭션 전파와 순서 합의 분리

// 특징:
// - Primary-Worker 아키텍처
// - DAG (Directed Acyclic Graph) 구조
// - 비동기 네트워크 환경에서 높은 처리량
// - Byzantine Fault Tolerant

// Tusk: Narwhal 위에서 동작하는 합의 프로토콜
// 목적: DAG에서 트랜잭션 순서 결정
// 특징: Zero-message overhead (추가 통신 불필요)
```

#### 1.1.2 Primary-Worker 구조

```rust
// Primary 노드: 합의 참여
// 소스: narwhal/primary/src/primary.rs

pub struct Primary {
    // 노드 식별자
    pub name: PublicKey,

    // 네트워크 커뮤니케이션
    network: P2pNetwork,

    // 현재 에포크의 committee (검증자 집합)
    committee: Committee,

    // Worker들과의 통신
    worker_channels: Vec<WorkerChannel>,

    // 헤더 스토어 (DAG 노드들)
    header_store: Store<HeaderDigest, Header>,

    // 인증서 스토어
    certificate_store: Store<CertificateDigest, Certificate>,

    // Core (합의 엔진)
    core: Core,
}

// Worker 노드: 트랜잭션 수신 및 배치 생성
// 소스: narwhal/worker/src/worker.rs

pub struct Worker {
    // Worker ID (Primary당 여러 Worker 가능)
    pub id: WorkerId,

    // Primary 주소
    primary_address: Multiaddr,

    // 트랜잭션 큐
    transactions: Receiver<Transaction>,

    // 배치 생성기
    batch_maker: BatchMaker,

    // 배치 스토어
    batch_store: Store<BatchDigest, Batch>,
}

// 배치 생성
impl BatchMaker {
    pub async fn make_batch(&mut self) -> Batch {
        // 목적: 트랜잭션들을 배치로 그룹화

        const MAX_BATCH_SIZE: usize = 500_000;  // 500KB
        const MAX_BATCH_DELAY: Duration = Duration::from_millis(100);

        let mut batch = Batch::default();
        let deadline = Instant::now() + MAX_BATCH_DELAY;

        // 1. 트랜잭션 수집 (크기 또는 시간 제한까지)
        loop {
            select! {
                tx = self.transactions.recv() => {
                    batch.transactions.push(tx);

                    // 크기 확인
                    if batch.size() >= MAX_BATCH_SIZE {
                        break;
                    }
                }
                _ = tokio::time::sleep_until(deadline) => {
                    // 타임아웃
                    break;
                }
            }
        }

        // 2. 배치 해시 계산
        batch.digest = Digest::new(&bincode::serialize(&batch)?);

        // 3. 스토어에 저장
        self.batch_store.write(batch.digest, batch.clone()).await?;

        batch
    }
}
```

#### 1.1.3 DAG 구조

```rust
// Header: DAG의 정점(vertex)
// 소스: narwhal/types/src/header.rs

#[derive(Clone, Serialize, Deserialize)]
pub struct Header {
    // 작성자 (Primary 노드)
    pub author: PublicKey,

    // 라운드 번호
    pub round: Round,

    // 이전 라운드 헤더들의 참조 (부모들)
    // 목적: DAG 구조 형성
    pub parents: BTreeSet<CertificateDigest>,

    // Worker들이 생성한 배치들
    // Worker ID -> Batch Digest
    pub payload: BTreeMap<WorkerId, BatchDigest>,

    // 타임스탬프
    pub created_at: SystemTime,

    // 작성자 서명
    pub signature: Signature,
}

// Certificate: 검증자들의 quorum 서명을 받은 헤더
#[derive(Clone, Serialize, Deserialize)]
pub struct Certificate {
    // 원본 헤더
    pub header: Header,

    // 검증자들의 서명 집합
    // 2/3+ 스테이크 필요
    pub aggregated_signature: AggregateSignature,

    // 서명한 검증자들
    pub signers: Vec<PublicKey>,
}

impl Header {
    // 헤더 생성 (Primary)
    pub fn new(
        author: PublicKey,
        round: Round,
        parents: BTreeSet<CertificateDigest>,
        payload: BTreeMap<WorkerId, BatchDigest>,
    ) -> Self {
        // 1. 헤더 구성
        let mut header = Header {
            author,
            round,
            parents,
            payload,
            created_at: SystemTime::now(),
            signature: Signature::default(),
        };

        // 2. 서명
        let digest = header.digest();
        header.signature = author.sign(&digest);

        header
    }

    pub fn digest(&self) -> Digest {
        // Bincode 직렬화 후 Blake2b 해시
        let bytes = bincode::serialize(self).unwrap();
        Digest::new(&bytes)
    }

    // 검증
    pub fn verify(&self, committee: &Committee) -> Result<()> {
        // 1. 서명 검증
        self.author.verify(&self.digest(), &self.signature)?;

        // 2. 부모 검증
        // - 정확히 2f+1개 부모 필요 (f = faulty nodes)
        // - 모든 부모는 이전 라운드 (round - 1)
        let quorum = committee.quorum_threshold();
        if self.parents.len() < quorum {
            return Err(Error::InsufficientParents);
        }

        Ok(())
    }
}
```

#### 1.1.4 헤더 전파 및 인증서 생성

```rust
// Primary의 합의 프로세스
impl Primary {
    // 헤더 생성 및 전파
    pub async fn propose_header(&mut self) {
        // === 1단계: 부모 선택 ===
        // 이전 라운드의 인증서들 중 2f+1개 선택
        let parents = self.select_parents().await;

        // === 2단계: Worker 배치 수집 ===
        let mut payload = BTreeMap::new();
        for worker in &self.workers {
            let batch_digest = worker.get_latest_batch().await?;
            payload.insert(worker.id, batch_digest);
        }

        // === 3단계: 헤더 생성 ===
        let header = Header::new(
            self.name,
            self.round,
            parents,
            payload,
        );

        // === 4단계: 다른 Primary들에게 전파 ===
        for peer in self.committee.primaries() {
            if peer != self.name {
                self.network.send(peer, Message::Header(header.clone())).await?;
            }
        }

        // === 5단계: 자신도 처리 ===
        self.process_header(header).await?;
    }

    // 헤더 수신 처리
    pub async fn process_header(&mut self, header: Header) -> Result<()> {
        // 1. 검증
        header.verify(&self.committee)?;

        // 2. 스토어에 저장
        self.header_store.write(header.digest(), header.clone()).await?;

        // 3. Vote 생성 및 전송
        let vote = Vote {
            digest: header.digest(),
            round: header.round,
            voter: self.name,
        };
        vote.sign(&self.secret_key);

        // 작성자에게 Vote 전송
        self.network.send(header.author, Message::Vote(vote)).await?;

        Ok(())
    }

    // Vote 수신 처리
    pub async fn process_vote(&mut self, vote: Vote) -> Result<()> {
        // 1. Vote 검증
        vote.verify()?;

        // 2. Vote 수집
        let votes = self.votes_aggregator.add_vote(vote);

        // 3. Quorum 확인 (2/3+ 스테이크)
        if votes.has_quorum(&self.committee) {
            // === Certificate 생성 ===
            let header = self.header_store.read(&vote.digest).await?;

            let certificate = Certificate {
                header: header.clone(),
                aggregated_signature: votes.aggregate_signatures(),
                signers: votes.signers(),
            };

            // 4. Certificate 저장
            self.certificate_store.write(
                certificate.digest(),
                certificate.clone(),
            ).await?;

            // 5. Certificate 전파
            for peer in self.committee.primaries() {
                self.network.send(
                    peer,
                    Message::Certificate(certificate.clone()),
                ).await?;
            }

            // 6. Core (합의 엔진)에 전달
            self.core.process_certificate(certificate).await?;
        }

        Ok(())
    }
}
```

#### 1.1.5 Tusk 합의 (DAG 순서화)

```rust
// Tusk: DAG에서 트랜잭션 순서 결정
// 소스: narwhal/consensus/src/tusk.rs

pub struct Tusk {
    // DAG
    dag: Dag,

    // 마지막 커밋된 라운드
    last_committed_round: Round,
}

impl Tusk {
    // 순서 결정 (commit)
    pub fn commit(&mut self) -> Vec<Certificate> {
        // === Tusk 알고리즘 ===
        // 목적: DAG를 선형 순서로 변환

        // 1. Anchor 선택
        // - 라운드 r에서 가장 많이 참조된 인증서
        // - 또는 결정론적 규칙 (예: 가장 낮은 해시)

        let anchor = self.select_anchor(self.last_committed_round + 2)?;

        // 2. Anchor로부터 역방향 순회
        // - Anchor의 조상들 수집
        // - 인과 관계 순서대로 정렬

        let mut committed = Vec::new();
        let mut to_visit = vec![anchor];
        let mut visited = HashSet::new();

        while let Some(cert) = to_visit.pop() {
            if visited.contains(&cert.digest()) {
                continue;
            }

            visited.insert(cert.digest());

            // 부모들 방문 큐에 추가
            for parent in &cert.header.parents {
                let parent_cert = self.dag.get(parent)?;
                to_visit.push(parent_cert);
            }

            // 커밋 리스트에 추가
            committed.push(cert);
        }

        // 3. 인과 순서로 정렬
        // - 부모가 항상 자식보다 먼저 오도록
        committed.sort_by_key(|c| c.header.round);

        // 4. 커밋 상태 업데이트
        self.last_committed_round = anchor.header.round;

        committed
    }

    fn select_anchor(&self, round: Round) -> Result<Certificate> {
        // Anchor 선택 규칙
        // 1. 라운드 r의 모든 인증서 가져오기
        let certs = self.dag.get_certificates_at_round(round);

        // 2. 라운드 r+1에서 가장 많이 참조된 것 선택
        let mut references = HashMap::new();
        for cert in self.dag.get_certificates_at_round(round + 1) {
            for parent in &cert.header.parents {
                *references.entry(parent).or_insert(0) += 1;
            }
        }

        // 3. 최다 참조 인증서
        certs.into_iter()
            .max_by_key(|c| references.get(&c.digest()).unwrap_or(&0))
            .ok_or(Error::NoAnchor)
    }
}
```

### 1.2 Sui 네트워크 프로토콜

**소스 위치**: `crates/sui-network/`

```rust
// Anemo: Sui의 P2P 네트워킹 라이브러리
// 소스: external-crates/anemo/

pub struct Network {
    // 로컬 피어 ID
    peer_id: PeerId,

    // 활성 연결들
    peers: HashMap<PeerId, Connection>,

    // 서비스 라우터
    router: Router,
}

// RPC 서비스 정의
pub trait NetworkService {
    // 트랜잭션 전파
    async fn submit_transaction(&self, tx: Transaction) -> Result<()>;

    // 객체 동기화
    async fn get_objects(&self, ids: Vec<ObjectID>) -> Result<Vec<Object>>;

    // 체크포인트 동기화
    async fn get_checkpoint(&self, seq: CheckpointSequenceNumber) -> Result<Checkpoint>;
}

// 연결 관리
impl Network {
    pub async fn connect(&mut self, peer: PeerId, addr: Multiaddr) -> Result<()> {
        // 1. TCP 연결
        let stream = TcpStream::connect(addr).await?;

        // 2. Noise 프로토콜 핸드셰이크 (암호화)
        let noise = NoiseConfig::new(&self.keypair);
        let conn = noise.handshake(stream).await?;

        // 3. 연결 저장
        self.peers.insert(peer, conn);

        Ok(())
    }

    // RPC 호출
    pub async fn rpc<Req, Res>(
        &self,
        peer: PeerId,
        request: Req,
    ) -> Result<Res> {
        // 1. 연결 가져오기
        let conn = self.peers.get(&peer)?;

        // 2. 요청 직렬화 (Bincode)
        let request_bytes = bincode::serialize(&request)?;

        // 3. 전송 및 응답 대기
        let response_bytes = conn.request(request_bytes).await?;

        // 4. 응답 역직렬화
        let response = bincode::deserialize(&response_bytes)?;

        Ok(response)
    }
}
```

---

## 2. 데이터베이스 레이어 (Database Layer)

### 2.1 RocksDB 기반 스토어

**소스 위치**: `crates/typed-store/`

#### 2.1.1 TypedStore
```rust
// TypedStore: 타입 안전 RocksDB 래퍼
// 소스: crates/typed-store/src/rocks/mod.rs

use rocksdb::{DB, Options, WriteBatch, ColumnFamily};

pub struct DBMap<K, V> {
    // RocksDB 인스턴스
    db: Arc<DB>,

    // Column Family 이름
    cf_name: String,

    // 팬텀 데이터 (타입 정보)
    _phantom: PhantomData<(K, V)>,
}

// 키-값 타입 제약
pub trait Key: Serialize + DeserializeOwned {}
pub trait Value: Serialize + DeserializeOwned {}

impl<K: Key, V: Value> DBMap<K, V> {
    // 삽입
    pub fn insert(&self, key: &K, value: &V) -> Result<()> {
        // 1. 직렬화 (Bincode)
        let key_bytes = bincode::serialize(key)?;
        let value_bytes = bincode::serialize(value)?;

        // 2. Column Family 가져오기
        let cf = self.db.cf_handle(&self.cf_name)
            .ok_or(Error::CFNotFound)?;

        // 3. RocksDB에 쓰기
        self.db.put_cf(cf, key_bytes, value_bytes)?;

        Ok(())
    }

    // 조회
    pub fn get(&self, key: &K) -> Result<Option<V>> {
        // 1. 키 직렬화
        let key_bytes = bincode::serialize(key)?;

        // 2. Column Family
        let cf = self.db.cf_handle(&self.cf_name)?;

        // 3. RocksDB에서 읽기
        match self.db.get_cf(cf, key_bytes)? {
            Some(value_bytes) => {
                // 4. 값 역직렬화
                let value = bincode::deserialize(&value_bytes)?;
                Ok(Some(value))
            }
            None => Ok(None),
        }
    }

    // 배치 쓰기
    pub fn multi_insert(
        &self,
        key_val_pairs: impl IntoIterator<Item = (K, V)>,
    ) -> Result<()> {
        // 원자적 배치 쓰기
        let mut batch = WriteBatch::default();
        let cf = self.db.cf_handle(&self.cf_name)?;

        for (key, value) in key_val_pairs {
            let key_bytes = bincode::serialize(&key)?;
            let value_bytes = bincode::serialize(&value)?;
            batch.put_cf(cf, key_bytes, value_bytes);
        }

        self.db.write(batch)?;
        Ok(())
    }

    // 반복자
    pub fn iter(&self) -> impl Iterator<Item = (K, V)> {
        let cf = self.db.cf_handle(&self.cf_name).unwrap();

        self.db
            .iterator_cf(cf, IteratorMode::Start)
            .map(|(k, v)| {
                let key: K = bincode::deserialize(&k).unwrap();
                let value: V = bincode::deserialize(&v).unwrap();
                (key, value)
            })
    }
}

// 사용 예제
pub struct SuiStore {
    // 객체 저장소: ObjectID -> Object
    objects: DBMap<ObjectID, Object>,

    // 트랜잭션 저장소: TransactionDigest -> Transaction
    transactions: DBMap<TransactionDigest, Transaction>,

    // 체크포인트 저장소: SequenceNumber -> Checkpoint
    checkpoints: DBMap<CheckpointSequenceNumber, Checkpoint>,
}
```

---

## 3. 스토리지 레이어 (Storage Layer)

### 3.1 객체 중심 스토리지

**소스 위치**: `crates/sui-core/src/authority/authority_store.rs`

#### 3.1.1 AuthorityStore
```rust
// AuthorityStore: Sui의 메인 저장소
// 목적: 객체, 트랜잭션, 체크포인트 관리

pub struct AuthorityStore {
    // === 객체 저장소 ===
    // ObjectID -> ObjectEntry
    // 목적: 최신 객체 버전 저장
    objects: DBMap<ObjectID, Object>,

    // 객체 버전 히스토리
    // (ObjectID, SequenceNumber) -> Object
    // 목적: 과거 버전 조회
    object_history: DBMap<(ObjectID, SequenceNumber), Object>,

    // === 트랜잭션 저장소 ===
    // TransactionDigest -> TransactionData
    transactions: DBMap<TransactionDigest, Transaction>,

    // TransactionDigest -> TransactionEffects
    // 목적: 실행 결과 저장
    effects: DBMap<TransactionDigest, TransactionEffects>,

    // === 소유권 인덱스 ===
    // Owner -> Vec<ObjectID>
    // 목적: 주소별 소유 객체 조회
    owner_index: DBMap<Owner, Vec<ObjectID>>,

    // === 체크포인트 ===
    // SequenceNumber -> Checkpoint
    checkpoints: DBMap<CheckpointSequenceNumber, Checkpoint>,

    // === 에포크 정보 ===
    // Epoch -> EpochInfo
    epochs: DBMap<EpochId, EpochInfo>,
}
```

#### 3.1.2 객체 모델

```rust
// Object: Sui의 기본 저장 단위
// 소스: crates/sui-types/src/object.rs

#[derive(Clone, Serialize, Deserialize)]
pub struct Object {
    // 객체 메타데이터
    pub data: Data,

    // 소유자 정보
    pub owner: Owner,

    // 이전 트랜잭션
    pub previous_transaction: TransactionDigest,

    // 스토리지 리베이트 (삭제시 환불)
    pub storage_rebate: u64,
}

#[derive(Clone, Serialize, Deserialize)]
pub struct ObjectID([u8; 32]);  // 256-bit 고유 ID

#[derive(Clone, Serialize, Deserialize)]
pub struct SequenceNumber(u64);  // 객체 버전

// 객체 데이터
#[derive(Clone, Serialize, Deserialize)]
pub enum Data {
    // Move 객체
    Move(MoveObject),

    // 패키지 (스마트 컨트랙트)
    Package(MovePackage),
}

#[derive(Clone, Serialize, Deserialize)]
pub struct MoveObject {
    // Move 타입 (예: "0x2::coin::Coin<0x2::sui::SUI>")
    pub type_: MoveObjectType,

    // 바이트코드로 인코딩된 필드 값들
    pub contents: Vec<u8>,

    // 버전
    pub version: SequenceNumber,
}

// 소유권 타입
#[derive(Clone, Serialize, Deserialize)]
pub enum Owner {
    // 주소 소유 (일반 사용자)
    AddressOwner(SuiAddress),

    // 객체 소유 (객체가 다른 객체 소유)
    // 용도: Child objects, Dynamic fields
    ObjectOwner(ObjectID),

    // 공유 객체 (누구나 접근 가능)
    // 용도: DEX pools, 투표 컨트랙트 등
    Shared {
        initial_shared_version: SequenceNumber,
    },

    // 불변 객체 (변경 불가)
    Immutable,
}
```

#### 3.1.3 객체 연산

```rust
impl AuthorityStore {
    // === 객체 조회 ===
    pub fn get_object(&self, id: &ObjectID) -> Result<Option<Object>> {
        // 최신 버전 조회
        self.objects.get(id)
    }

    pub fn get_object_by_version(
        &self,
        id: &ObjectID,
        version: SequenceNumber,
    ) -> Result<Option<Object>> {
        // 특정 버전 조회
        self.object_history.get(&(*id, version))
    }

    // === 객체 업데이트 ===
    pub fn update_objects(
        &self,
        written_objects: Vec<Object>,
        deleted_objects: Vec<ObjectID>,
    ) -> Result<()> {
        // 배치 쓰기
        let mut batch = WriteBatch::default();

        // 1. 새 객체 / 업데이트된 객체
        for obj in written_objects {
            let id = obj.id();
            let version = obj.version();

            // 최신 버전 업데이트
            self.objects.insert(&id, &obj)?;

            // 히스토리에 추가
            self.object_history.insert(&(id, version), &obj)?;

            // 소유권 인덱스 업데이트
            self.update_owner_index(&obj)?;
        }

        // 2. 삭제된 객체
        for id in deleted_objects {
            // 최신 버전에서 제거 (히스토리는 유지)
            self.objects.remove(&id)?;

            // 소유권 인덱스에서 제거
            self.remove_from_owner_index(&id)?;
        }

        Ok(())
    }

    // === 소유권 인덱스 ===
    fn update_owner_index(&self, obj: &Object) -> Result<()> {
        // 목적: 주소별 소유 객체 빠른 조회

        let owner = obj.owner;
        let id = obj.id();

        // 기존 리스트 가져오기
        let mut objects = self.owner_index
            .get(&owner)?
            .unwrap_or_default();

        // 객체 추가 (중복 제거)
        if !objects.contains(&id) {
            objects.push(id);
        }

        // 업데이트
        self.owner_index.insert(&owner, &objects)?;

        Ok(())
    }

    // === 주소의 모든 객체 조회 ===
    pub fn get_owned_objects(&self, owner: &Owner) -> Result<Vec<Object>> {
        // 1. 소유권 인덱스에서 ObjectID 리스트 조회
        let object_ids = self.owner_index.get(owner)?
            .unwrap_or_default();

        // 2. 각 ObjectID로 객체 로드
        let mut objects = Vec::new();
        for id in object_ids {
            if let Some(obj) = self.objects.get(&id)? {
                objects.push(obj);
            }
        }

        Ok(objects)
    }
}
```

### 3.2 트랜잭션 및 Effects

```rust
// TransactionData: 트랜잭션 입력
// 소스: crates/sui-types/src/transaction.rs

#[derive(Clone, Serialize, Deserialize)]
pub struct TransactionData {
    // 트랜잭션 종류
    pub kind: TransactionKind,

    // 발신자
    pub sender: SuiAddress,

    // Gas 정보
    pub gas_data: GasData,

    // 만료 시간
    pub expiration: TransactionExpiration,
}

#[derive(Clone, Serialize, Deserialize)]
pub enum TransactionKind {
    // Move 함수 호출
    ProgrammableTransaction(ProgrammableTransaction),

    // 시스템 트랜잭션
    ChangeEpoch(ChangeEpoch),
    Genesis(Genesis),
    ConsensusCommitPrologue(ConsensusCommitPrologue),
}

#[derive(Clone, Serialize, Deserialize)]
pub struct ProgrammableTransaction {
    // 입력 객체들 및 순수 값들
    pub inputs: Vec<CallArg>,

    // 실행할 명령들 (Move 함수 호출 체인)
    pub commands: Vec<Command>,
}

#[derive(Clone, Serialize, Deserialize)]
pub enum Command {
    // Move 함수 호출
    MoveCall(MoveCall),

    // 객체 전송
    TransferObjects(Vec<Argument>, Argument),

    // 객체 분할 (Coin)
    SplitCoins(Argument, Vec<Argument>),

    // 객체 병합
    MergeCoins(Argument, Vec<Argument>),

    // 패키지 발행
    Publish(Vec<Vec<u8>>, Vec<ObjectID>),

    // ... 기타
}

// TransactionEffects: 실행 결과
#[derive(Clone, Serialize, Deserialize)]
pub struct TransactionEffects {
    // 실행 상태
    pub status: ExecutionStatus,

    // Gas 사용량
    pub gas_used: GasCostSummary,

    // 생성된 객체들
    pub created: Vec<(ObjectRef, Owner)>,

    // 수정된 객체들
    pub mutated: Vec<(ObjectRef, Owner)>,

    // 삭제된 객체들
    pub deleted: Vec<ObjectRef>,

    // 발생한 이벤트들
    pub events: Vec<Event>,

    // 의존성 (이 트랜잭션이 읽은 객체들)
    pub dependencies: Vec<TransactionDigest>,
}

// 객체 참조 (ID + 버전 + 다이제스트)
pub type ObjectRef = (ObjectID, SequenceNumber, ObjectDigest);
```

### 3.3 스냅샷 및 Pruning

```rust
// 스냅샷: 특정 체크포인트의 전체 상태
// 소스: crates/sui-core/src/checkpoints/

impl AuthorityStore {
    // 스냅샷 생성
    pub fn create_snapshot(&self, checkpoint: CheckpointSequenceNumber) -> Result<()> {
        // 목적: 빠른 동기화 및 백업

        let snapshot_dir = self.config.snapshot_path
            .join(format!("checkpoint_{}", checkpoint));

        // 1. RocksDB 체크포인트 생성 (하드링크)
        // 장점: 디스크 공간 효율적
        self.db.create_checkpoint(&snapshot_dir)?;

        // 2. 메타데이터 파일 생성
        let metadata = SnapshotMetadata {
            checkpoint_seq: checkpoint,
            epoch: self.get_epoch_at_checkpoint(checkpoint)?,
            total_objects: self.count_objects()?,
            timestamp: SystemTime::now(),
        };

        let metadata_file = snapshot_dir.join("MANIFEST");
        serde_json::to_writer(File::create(metadata_file)?, &metadata)?;

        // 3. 압축 (옵션)
        if self.config.compress_snapshots {
            Self::compress_snapshot(&snapshot_dir)?;
        }

        Ok(())
    }

    // Pruning: 오래된 객체 버전 제거
    pub fn prune_objects(&self, retention_epochs: EpochId) -> Result<()> {
        // 목적: 디스크 공간 절약

        // num-epochs-to-retain 설정
        // - 0: 공격적 (최신만 유지) - 읽기 쿼리 불가
        // - N: N 에포크 이전까지 유지

        let current_epoch = self.get_current_epoch()?;
        let prune_before_epoch = current_epoch.saturating_sub(retention_epochs);

        // 1. 에포크 경계 체크포인트 찾기
        let prune_checkpoint = self.get_epoch_start_checkpoint(prune_before_epoch)?;

        // 2. 오래된 객체 버전 제거
        for (obj_id, version) in self.object_history.iter() {
            if version < prune_checkpoint.into() {
                // 히스토리에서 제거
                self.object_history.remove(&(obj_id, version))?;
            }
        }

        // 3. RocksDB 압축 트리거
        // 목적: 실제 디스크 공간 회수
        self.db.compact_range(None::<&[u8]>, None::<&[u8]>);

        Ok(())
    }
}
```

---

## 4. RPC 레이어 (RPC Layer)

### 4.1 JSON-RPC 서버

**소스 위치**: `crates/sui-json-rpc/`

#### 4.1.1 서버 구조
```rust
// JsonRpcServer: Sui의 JSON-RPC 서버
// 소스: crates/sui-json-rpc/src/lib.rs

use jsonrpsee::{
    core::RpcResult,
    proc_macros::rpc,
    server::{Server, ServerBuilder},
};

// RPC API 트레잇
#[rpc(server, client, namespace = "sui")]
pub trait SuiRpcApi {
    // === 객체 조회 ===
    #[method(name = "getObject")]
    async fn get_object(
        &self,
        object_id: ObjectID,
        options: Option<SuiObjectDataOptions>,
    ) -> RpcResult<SuiObjectResponse>;

    #[method(name = "multiGetObjects")]
    async fn multi_get_objects(
        &self,
        object_ids: Vec<ObjectID>,
        options: Option<SuiObjectDataOptions>,
    ) -> RpcResult<Vec<SuiObjectResponse>>;

    // === 계정 조회 ===
    #[method(name = "getOwnedObjects")]
    async fn get_owned_objects(
        &self,
        address: SuiAddress,
        query: Option<SuiObjectResponseQuery>,
        cursor: Option<ObjectID>,
        limit: Option<usize>,
    ) -> RpcResult<PaginatedObjectsResponse>;

    // === 트랜잭션 실행 ===
    #[method(name = "dryRunTransactionBlock")]
    async fn dry_run_transaction_block(
        &self,
        tx_bytes: Base64,
    ) -> RpcResult<DryRunTransactionBlockResponse>;

    #[method(name = "executeTransactionBlock")]
    async fn execute_transaction_block(
        &self,
        tx_bytes: Base64,
        signatures: Vec<Base64>,
        options: Option<SuiTransactionBlockResponseOptions>,
        request_type: Option<ExecuteTransactionRequestType>,
    ) -> RpcResult<SuiTransactionBlockResponse>;

    // === 트랜잭션 조회 ===
    #[method(name = "getTransactionBlock")]
    async fn get_transaction_block(
        &self,
        digest: TransactionDigest,
        options: Option<SuiTransactionBlockResponseOptions>,
    ) -> RpcResult<SuiTransactionBlockResponse>;

    // === 체크포인트 조회 ===
    #[method(name = "getCheckpoint")]
    async fn get_checkpoint(
        &self,
        id: CheckpointId,
    ) -> RpcResult<Checkpoint>;

    // ... 100+ 메서드
}
```

#### 4.1.2 주요 RPC 구현

```rust
// SuiRpcModule: RPC API 구현
pub struct SuiRpcModule {
    // 상태 접근
    state: Arc<AuthorityState>,

    // 트랜잭션 오케스트레이터
    transaction_orchestrator: Arc<TransactionOrchestrator>,
}

impl SuiRpcApiServer for SuiRpcModule {
    // 객체 조회
    async fn get_object(
        &self,
        object_id: ObjectID,
        options: Option<SuiObjectDataOptions>,
    ) -> RpcResult<SuiObjectResponse> {
        // 1. 스토어에서 객체 로드
        let object = self.state.database
            .get_object(&object_id)
            .map_err(|e| internal_error(e))?;

        // 2. 응답 형식으로 변환
        let response = match object {
            Some(obj) => {
                // 옵션에 따라 상세 정보 포함
                let data = if options.show_content {
                    Some(SuiParsedData::try_from_object(obj.data.clone())?)
                } else {
                    None
                };

                let owner = if options.show_owner {
                    Some(obj.owner)
                } else {
                    None
                };

                SuiObjectResponse {
                    data: Some(SuiObjectData {
                        object_id: obj.id(),
                        version: obj.version(),
                        digest: obj.digest(),
                        type_: obj.type_().map(|t| t.to_string()),
                        owner,
                        previous_transaction: Some(obj.previous_transaction),
                        storage_rebate: Some(obj.storage_rebate),
                        content: data,
                    }),
                    error: None,
                }
            }
            None => {
                SuiObjectResponse {
                    data: None,
                    error: Some(SuiObjectResponseError::NotExists { object_id }),
                }
            }
        };

        Ok(response)
    }

    // 트랜잭션 실행
    async fn execute_transaction_block(
        &self,
        tx_bytes: Base64,
        signatures: Vec<Base64>,
        options: Option<SuiTransactionBlockResponseOptions>,
        request_type: Option<ExecuteTransactionRequestType>,
    ) -> RpcResult<SuiTransactionBlockResponse> {
        // 1. 트랜잭션 역직렬화
        let tx_data: TransactionData = bcs::from_bytes(&tx_bytes.to_vec()?)?;

        // 2. 서명 복원
        let sigs: Vec<Signature> = signatures
            .into_iter()
            .map(|s| Signature::from_bytes(&s.to_vec()?))
            .collect::<Result<_, _>>()?;

        // 3. SignedTransaction 생성
        let signed_tx = SignedTransaction::new(
            tx_data,
            sigs,
        );

        // 4. 서명 검증
        signed_tx.verify()?;

        // 5. 트랜잭션 제출
        let response = match request_type {
            // 즉시 실행 후 결과 대기
            Some(ExecuteTransactionRequestType::WaitForLocalExecution) => {
                let effects = self.transaction_orchestrator
                    .execute_transaction_block(signed_tx)
                    .await?;

                // 상세 응답 생성
                self.create_transaction_response(
                    effects.transaction_digest(),
                    options,
                )
                .await?
            }

            // 제출만 (비동기)
            _ => {
                self.transaction_orchestrator
                    .submit_transaction(signed_tx)
                    .await?;

                // 다이제스트만 반환
                SuiTransactionBlockResponse {
                    digest: signed_tx.digest(),
                    ..Default::default()
                }
            }
        };

        Ok(response)
    }

    // Dry run (시뮬레이션)
    async fn dry_run_transaction_block(
        &self,
        tx_bytes: Base64,
    ) -> RpcResult<DryRunTransactionBlockResponse> {
        // 목적: 실제 실행 없이 결과 예측

        // 1. 트랜잭션 파싱
        let tx_data: TransactionData = bcs::from_bytes(&tx_bytes.to_vec()?)?;

        // 2. 임시 상태 생성 (현재 상태의 복사본)
        let temp_store = self.state.database.clone_for_simulation();

        // 3. 시뮬레이션 실행
        let (effects, return_values, error) = self.state
            .dry_exec_transaction(tx_data, temp_store)
            .await?;

        // 4. 결과 반환 (상태는 실제로 변경되지 않음)
        Ok(DryRunTransactionBlockResponse {
            effects,
            events: effects.events().clone(),
            object_changes: self.compute_object_changes(&effects)?,
            balance_changes: self.compute_balance_changes(&effects)?,
            input: return_values,
            error,
        })
    }

    // 소유 객체 조회 (페이지네이션)
    async fn get_owned_objects(
        &self,
        address: SuiAddress,
        query: Option<SuiObjectResponseQuery>,
        cursor: Option<ObjectID>,
        limit: Option<usize>,
    ) -> RpcResult<PaginatedObjectsResponse> {
        const MAX_LIMIT: usize = 50;
        let limit = limit.unwrap_or(MAX_LIMIT).min(MAX_LIMIT);

        // 1. 소유권 인덱스 조회
        let owner = Owner::AddressOwner(address);
        let mut objects = self.state.database
            .get_owned_objects(&owner)?;

        // 2. 필터 적용
        if let Some(query) = query {
            objects.retain(|obj| query.matches(obj));
        }

        // 3. 정렬 (ObjectID 순)
        objects.sort_by_key(|obj| obj.id());

        // 4. 커서 이후부터 선택
        let start = cursor
            .and_then(|c| objects.iter().position(|obj| obj.id() == c))
            .map(|pos| pos + 1)
            .unwrap_or(0);

        // 5. 페이지 추출
        let page = &objects[start..][..limit.min(objects.len() - start)];

        // 6. 응답 생성
        let data: Vec<_> = page
            .iter()
            .map(|obj| self.object_to_response(obj))
            .collect();

        let next_cursor = page.last().map(|obj| obj.id());
        let has_next_page = start + limit < objects.len();

        Ok(PaginatedObjectsResponse {
            data,
            next_cursor,
            has_next_page,
        })
    }
}
```

### 4.2 WebSocket (구독)

```rust
// PubSub API
#[rpc(server, client, namespace = "suix")]
pub trait SuiSubscriptionApi {
    // 이벤트 구독
    #[subscription(name = "subscribeEvent", item = SuiEvent)]
    async fn subscribe_event(
        &self,
        filter: EventFilter,
    ) -> SubscriptionResult;

    // 트랜잭션 구독
    #[subscription(name = "subscribeTransaction", item = SuiTransactionBlockEffects)]
    async fn subscribe_transaction(
        &self,
        filter: TransactionFilter,
    ) -> SubscriptionResult;
}

// 이벤트 발행
impl AuthorityState {
    pub fn notify_transaction_executed(
        &self,
        effects: &TransactionEffects,
    ) {
        // 1. 이벤트 추출
        for event in &effects.events {
            // 2. 구독자 필터 매칭
            for subscriber in self.event_subscribers.iter() {
                if subscriber.filter.matches(event) {
                    // 3. WebSocket으로 이벤트 전송
                    subscriber.send(event.clone());
                }
            }
        }
    }
}
```

---

## 5. 핵심 아키텍처 컴포넌트

### 5.1 Move VM 통합

**소스 위치**: `crates/sui-framework/`, `external-crates/move/`

```rust
// Move VM: Sui의 스마트 컨트랙트 실행 엔진
// 소스: external-crates/move/move-vm/runtime/src/

use move_vm_runtime::MoveVM;
use move_binary_format::CompiledModule;

pub struct SuiMoveVM {
    // Move VM 인스턴스
    vm: MoveVM,

    // 네이티브 함수들
    natives: NativeFunctionTable,
}

impl SuiMoveVM {
    // Move 함수 실행
    pub fn execute_function(
        &self,
        session: &mut Session,
        module_id: &ModuleId,
        function_name: &Identifier,
        type_args: Vec<TypeTag>,
        args: Vec<Vec<u8>>,  // BCS 인코딩된 인자들
        gas_budget: u64,
    ) -> VMResult<Vec<Vec<u8>>> {  // 반환값들
        // 1. 함수 시그니처 검증
        let function = session
            .load_function(module_id, function_name, &type_args)?;

        // 2. 인자 역직렬화
        let deserialized_args = function.parameters
            .iter()
            .zip(args.iter())
            .map(|(ty, arg)| {
                session.deserialize_value(ty, arg)
            })
            .collect::<Result<Vec<_>, _>>()?;

        // 3. 함수 실행
        let return_values = session.execute_function_bypass_visibility(
            module_id,
            function_name,
            type_args,
            deserialized_args,
            &mut Gas::new(gas_budget),
        )?;

        // 4. 반환값 직렬화
        let serialized_returns = return_values
            .into_iter()
            .map(|v| v.simple_serialize())
            .collect::<Option<Vec<_>>>()
            .ok_or(VMError::SerializationError)?;

        Ok(serialized_returns)
    }
}

// 트랜잭션 실행 엔트리포인트
pub fn execute_transaction(
    vm: &SuiMoveVM,
    state: &mut AuthorityState,
    tx: TransactionData,
) -> Result<TransactionEffects> {
    // === 1단계: 입력 객체 로드 ===
    let mut input_objects = Vec::new();
    for input in &tx.input_objects() {
        let obj = state.get_object(input)?;
        input_objects.push(obj);
    }

    // === 2단계: Move 세션 시작 ===
    let mut session = vm.new_session(&state.database);

    // === 3단계: Programmable Transaction 실행 ===
    match tx.kind {
        TransactionKind::ProgrammableTransaction(pt) => {
            // 3.1. 입력 준비
            let mut results = Vec::new();
            for input in &pt.inputs {
                match input {
                    CallArg::Pure(bytes) => {
                        results.push(bytes.clone());
                    }
                    CallArg::Object(obj_ref) => {
                        let obj = input_objects.iter()
                            .find(|o| o.id() == obj_ref.0)
                            .unwrap();
                        results.push(bcs::to_bytes(obj)?);
                    }
                }
            }

            // 3.2. 명령 순차 실행
            for command in &pt.commands {
                match command {
                    Command::MoveCall(call) => {
                        // Move 함수 호출
                        let args = call.arguments.iter()
                            .map(|arg| {
                                match arg {
                                    Argument::Input(i) => results[*i].clone(),
                                    Argument::Result(i) => results[*i].clone(),
                                    // ...
                                }
                            })
                            .collect();

                        let return_values = vm.execute_function(
                            &mut session,
                            &call.package,
                            &call.module,
                            &call.function,
                            call.type_arguments.clone(),
                            args,
                            tx.gas_data.budget,
                        )?;

                        results.extend(return_values);
                    }

                    Command::TransferObjects(objects, recipient) => {
                        // 객체 전송
                        for obj_arg in objects {
                            let obj_bytes = &results[obj_arg.input_index()];
                            let mut obj: Object = bcs::from_bytes(obj_bytes)?;

                            // 소유자 변경
                            let recipient_addr = bcs::from_bytes(&results[recipient.input_index()])?;
                            obj.owner = Owner::AddressOwner(recipient_addr);

                            // 변경 기록
                            session.mark_mutated(obj);
                        }
                    }

                    // ... 기타 명령들
                }
            }
        }
        // ... 기타 트랜잭션 타입
    }

    // === 4단계: 세션 완료 및 변경 사항 추출 ===
    let (changeset, events) = session.finish()?;

    // === 5단계: Effects 생성 ===
    let mut created = Vec::new();
    let mut mutated = Vec::new();
    let mut deleted = Vec::new();

    for (obj_id, change) in changeset.objects {
        match change {
            ObjectChange::Write(obj) => {
                if obj.is_new() {
                    created.push((obj.compute_object_reference(), obj.owner));
                } else {
                    mutated.push((obj.compute_object_reference(), obj.owner));
                }
                // 상태에 저장
                state.database.update_object(obj)?;
            }
            ObjectChange::Delete => {
                deleted.push(obj_id);
                state.database.delete_object(&obj_id)?;
            }
        }
    }

    let effects = TransactionEffects {
        status: ExecutionStatus::Success,
        gas_used: changeset.gas_used,
        created,
        mutated,
        deleted,
        events,
        dependencies: tx.dependencies(),
    };

    Ok(effects)
}
```

### 5.2 객체 중심 병렬 실행

```rust
// Sui의 핵심 혁신: 객체 소유권 기반 병렬 실행
// 목적: 충돌 없는 트랜잭션 동시 처리

pub struct TransactionManager {
    // 객체 잠금 테이블
    // ObjectID -> Transaction in progress
    locks: DashMap<ObjectID, TransactionDigest>,
}

impl TransactionManager {
    // 트랜잭션 실행 가능 여부 확인
    pub fn can_execute(&self, tx: &Transaction) -> bool {
        // 1. 트랜잭션이 사용하는 객체들 추출
        let input_objects = tx.input_objects();

        // 2. 모든 객체가 잠기지 않았는지 확인
        for obj_id in input_objects {
            if self.locks.contains_key(&obj_id) {
                return false;  // 충돌!
            }
        }

        true  // 병렬 실행 가능!
    }

    // 객체 잠금
    pub fn lock_objects(&self, tx_digest: TransactionDigest, objects: &[ObjectID]) {
        for obj_id in objects {
            self.locks.insert(*obj_id, tx_digest);
        }
    }

    // 객체 해제
    pub fn unlock_objects(&self, objects: &[ObjectID]) {
        for obj_id in objects {
            self.locks.remove(obj_id);
        }
    }
}

// 병렬 실행 엔진
pub async fn execute_transactions_parallel(
    txs: Vec<Transaction>,
) -> Vec<TransactionEffects> {
    // 1. 실행 가능한 트랜잭션 필터링
    let mut executable = Vec::new();
    let mut waiting = VecDeque::from(txs);

    while let Some(tx) = waiting.pop_front() {
        if can_execute(&tx) {
            executable.push(tx);
        } else {
            waiting.push_back(tx);  // 나중에 재시도
        }
    }

    // 2. 병렬 실행
    let handles: Vec<_> = executable
        .into_iter()
        .map(|tx| {
            tokio::spawn(async move {
                // 각 트랜잭션을 별도 task에서 실행
                execute_transaction(tx).await
            })
        })
        .collect();

    // 3. 결과 수집
    let mut results = Vec::new();
    for handle in handles {
        results.push(handle.await.unwrap());
    }

    results
}

// 단순 트랜잭션 최적화 (FastPath)
// - 소유 객체만 사용 (공유 객체 없음)
// - 합의 불필요!
// - 검증자가 독립적으로 즉시 처리

pub fn is_simple_transaction(tx: &Transaction) -> bool {
    // 모든 입력 객체가 소유 객체인지 확인
    tx.input_objects().iter().all(|obj_ref| {
        let obj = get_object(obj_ref);
        matches!(obj.owner, Owner::AddressOwner(_) | Owner::ObjectOwner(_))
    })
}

// FastPath 실행
pub async fn execute_simple_transaction(tx: Transaction) -> TransactionEffects {
    // 1. 서명 검증
    tx.verify_signature()?;

    // 2. 객체 소유권 검증
    verify_ownership(&tx)?;

    // 3. 즉시 실행 (합의 스킵!)
    let effects = execute_transaction(tx)?;

    // 4. 인증서 발급
    let cert = Certificate {
        transaction: tx,
        effects,
        signatures: vec![self.sign(&effects)],
    };

    // 5. 클라이언트에 즉시 반환
    Ok(effects)
}
```

### 5.3 Checkpoints & Epochs

```rust
// Checkpoint: 일정 트랜잭션을 묶은 배치
// 목적: Finality 제공, 동기화 단위

#[derive(Clone, Serialize, Deserialize)]
pub struct Checkpoint {
    // 시퀀스 번호
    pub sequence_number: CheckpointSequenceNumber,

    // 에포크
    pub epoch: EpochId,

    // 포함된 트랜잭션들
    pub transactions: Vec<TransactionDigest>,

    // 이전 체크포인트 다이제스트
    pub previous_digest: Option<CheckpointDigest>,

    // 에포크 롤링 가스 비용 요약
    pub epoch_rolling_gas_cost_summary: GasCostSummary,

    // 타임스탬프
    pub timestamp_ms: u64,

    // 검증자 서명 (2/3+ 스테이크)
    pub validator_signature: AggregateAuthoritySignature,
}

// 체크포인트 생성 (검증자)
impl CheckpointBuilder {
    pub async fn build_checkpoint(&mut self) -> Result<Checkpoint> {
        const TRANSACTIONS_PER_CHECKPOINT: usize = 200;

        // 1. 실행 완료된 트랜잭션 수집
        let mut transactions = Vec::new();
        while transactions.len() < TRANSACTIONS_PER_CHECKPOINT {
            if let Some(tx_digest) = self.pending_transactions.pop() {
                transactions.push(tx_digest);
            } else {
                break;  // 더 이상 트랜잭션 없음
            }
        }

        // 2. 체크포인트 생성
        let checkpoint = Checkpoint {
            sequence_number: self.next_checkpoint_seq,
            epoch: self.current_epoch,
            transactions,
            previous_digest: self.last_checkpoint_digest,
            epoch_rolling_gas_cost_summary: self.compute_gas_summary(),
            timestamp_ms: SystemTime::now().duration_since(UNIX_EPOCH)?.as_millis() as u64,
            validator_signature: AggregateAuthoritySignature::default(),  // 아직 서명 없음
        };

        // 3. 서명 생성
        let signature = self.authority_key.sign(&checkpoint.digest());

        // 4. 다른 검증자들에게 전파하여 서명 수집
        // (Narwhal을 통한 합의)

        Ok(checkpoint)
    }
}

// Epoch: 검증자 집합이 고정된 기간
#[derive(Clone, Serialize, Deserialize)]
pub struct EpochInfo {
    pub epoch: EpochId,

    // 검증자 집합 및 스테이크
    pub validators: Vec<ValidatorMetadata>,

    // 에포크 시작 체크포인트
    pub epoch_start_checkpoint: CheckpointSequenceNumber,

    // 에포크 종료 체크포인트
    pub epoch_end_checkpoint: Option<CheckpointSequenceNumber>,

    // 프로토콜 버전
    pub protocol_version: ProtocolVersion,

    // 시스템 상태 (총 스테이크, 스토리지 펀드 등)
    pub system_state: SystemState,
}

// Epoch 전환
impl EpochManager {
    pub async fn end_epoch(&mut self) -> Result<()> {
        // 1. 현재 에포크 종료
        let current_epoch = self.current_epoch_info();

        // 2. 스테이크 재계산
        // - 검증자 보상 분배
        // - 새로운 스테이크 델리게이션 적용
        let new_validators = self.compute_next_epoch_validators()?;

        // 3. 새 에포크 시작
        let next_epoch = EpochInfo {
            epoch: current_epoch.epoch + 1,
            validators: new_validators,
            epoch_start_checkpoint: self.next_checkpoint_seq,
            epoch_end_checkpoint: None,
            protocol_version: self.next_protocol_version(),
            system_state: self.compute_next_system_state()?,
        };

        // 4. 에포크 정보 저장
        self.store.insert_epoch_info(&next_epoch)?;

        // 5. Narwhal committee 업데이트
        self.narwhal.reconfigure(next_epoch.committee())?;

        Ok(())
    }
}
```

---

## 학습 로드맵

### Phase 1: 기초 이해 (1-2주)
1. **객체 모델**
   - Sui의 객체 중심 설계 이해
   - 소유권 타입 (Owned, Shared, Immutable)
   - Object vs Account 차이점

2. **Move 프로그래밍**
   - Move 언어 기초
   - Sui Move 확장 (object, transfer)
   - 간단한 컨트랙트 작성

### Phase 2: 심화 학습 (2-3주)
1. **Narwhal & Tusk**
   - DAG 구조 이해
   - Primary-Worker 아키텍처
   - 합의 프로세스

2. **스토리지 & 인덱싱**
   - RocksDB 스키마
   - 객체 버전 관리
   - 소유권 인덱스

### Phase 3: 전문가 (3-4주)
1. **병렬 실행**
   - FastPath vs ConsensusPath
   - 트랜잭션 충돌 감지
   - 성능 최적화

2. **Checkpoints & Epochs**
   - 체크포인트 생성 과정
   - Finality 메커니즘
   - 에포크 전환

### 실습 프로젝트
1. Move 스마트 컨트랙트 (NFT, DeFi)
2. 객체 소유권 추적 도구
3. 체크포인트 검증기
4. 트랜잭션 시뮬레이터

---

## 추가 학습 자료

### 공식 문서
- Sui 문서: https://docs.sui.io
- Move Book: https://move-book.com
- Sui Move by Example: https://examples.sui.io

### 소스 코드
- Sui 저장소: https://github.com/MystenLabs/sui
- 주요 디렉토리:
  - `crates/sui-core/`: 핵심 로직
  - `crates/sui-types/`: 데이터 타입
  - `crates/sui-json-rpc/`: RPC 서버
  - `crates/sui-framework/`: Move 프레임워크
  - `narwhal/`: 합의 엔진

### 개발 도구
```bash
# Sui CLI
sui client

# Move 컴파일
sui move build

# 로컬 네트워크
sui start

# RPC 호출
sui client call --function <function> --module <module> --package <package>
```

## 특별 주제: Sui vs 다른 블록체인

### 객체 vs 계정 모델
- **Ethereum**: 계정 기반, 글로벌 상태
- **Sui**: 객체 기반, 소유권 명확

### 병렬 실행
- **Ethereum**: 순차 실행
- **Solana**: Sealevel (계정 충돌 기반)
- **Sui**: 객체 소유권 기반 (가장 직관적!)

### 합의
- **Ethereum**: Gasper (PoS)
- **Solana**: Tower BFT + PoH
- **Sui**: Narwhal-Tusk (DAG) + FastPath (소유 객체는 합의 불필요!)
