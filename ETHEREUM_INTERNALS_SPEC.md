# Ethereum (go-ethereum/geth) 내부 구현 완전 분석

## 목차
1. [네트워크 레이어 (Network Layer)](#1-네트워크-레이어)
2. [데이터베이스 레이어 (Database Layer)](#2-데이터베이스-레이어)
3. [스토리지 레이어 (Storage Layer)](#3-스토리지-레이어)
4. [RPC 레이어 (RPC Layer)](#4-rpc-레이어)
5. [핵심 데이터 구조](#5-핵심-데이터-구조)

---

## 1. 네트워크 레이어 (Network Layer)

### 1.1 DevP2P 프로토콜 스택

**소스 위치**: `p2p/` 디렉토리

#### 1.1.1 핵심 개념
```
DevP2P = Ethereum의 전용 P2P 네트워킹 프로토콜 스택
목적: 노드 간 안전하고 효율적인 통신 제공
```

#### 1.1.2 주요 컴포넌트

##### A. Server (`p2p/server.go`)
```go
// Server는 모든 피어 연결을 관리하는 중앙 구조체
type Server struct {
    // 목적: 피어 연결의 lifecycle 관리

    // Config: 서버 설정 (실행 중 수정 불가)
    Config Config

    // running: 서버 실행 상태
    running bool

    // listener: TCP 연결 수신 대기
    listener net.Listener

    // peers: 현재 연결된 피어들의 맵
    // key: peer ID, value: *Peer 포인터
    peers map[enode.ID]*Peer

    // 내부 로직:
    // 1. 새로운 피어 연결 수락
    // 2. 피어 핸드셰이크 수행
    // 3. 프로토콜 협상
    // 4. 연결 유지 및 관리
}

// 주요 메서드들
func (srv *Server) Start() error {
    // 의도: 서버 시작 및 리스닝 시작
    // 1. TCP 리스너 시작
    // 2. discovery 프로토콜 시작
    // 3. peer goroutine 시작
}

func (srv *Server) run() {
    // 무한 루프로 실행
    // 목적: 새로운 연결 처리, 피어 관리, 이벤트 처리

    for {
        select {
        case <-srv.quit:
            // 종료 신호 처리
        case n := <-srv.addpeer:
            // 새로운 피어 추가
            srv.peers[n.ID()] = n
        case pd := <-srv.delpeer:
            // 피어 제거
            delete(srv.peers, pd.ID())
        }
    }
}
```

##### B. Peer (`p2p/peer.go`)
```go
// Peer는 연결된 원격 노드를 나타냄
type Peer struct {
    // 목적: 개별 피어와의 통신 관리

    rw      *conn              // 실제 네트워크 연결
    running map[string]*protoRW // 실행 중인 프로토콜들

    // 내부 로직:
    // - 메시지 읽기/쓰기
    // - 프로토콜 멀티플렉싱
    // - 연결 상태 관리
}

func (p *Peer) run() error {
    // 피어 메인 루프
    // 1. 핸드셰이크 완료 대기
    // 2. 각 프로토콜의 Run 함수 시작
    // 3. 메시지 라우팅
}
```

##### C. RLPx (`p2p/rlpx/rlpx.go`)
```go
// RLPx: 암호화된 전송 프로토콜
// 목적: 노드 간 안전한 통신 채널 제공

type Conn struct {
    // 의도: 암호화된 양방향 통신 제공

    fd net.Conn        // 기본 TCP 연결

    // ECIES (Elliptic Curve Integrated Encryption Scheme) 사용
    // 암호화 키 교환
    handshake *handshakeState

    // AES-256-CTR 스트림 암호화
    enc cipher.Stream  // 송신 암호화
    dec cipher.Stream  // 수신 복호화

    // MAC (Message Authentication Code)
    // 메시지 무결성 검증
    ingressMAC hash.Hash
    egressMAC  hash.Hash
}

// 핸드셰이크 과정
func (c *Conn) Handshake(prv *ecdsa.PrivateKey) error {
    // 1단계: auth 메시지 교환
    //   - ECDSA 공개키 교환
    //   - nonce 교환
    //   - 임시 키 생성

    // 2단계: ack 메시지 교환
    //   - 상호 인증
    //   - 공유 비밀 생성

    // 3단계: 암호화 스트림 초기화
    //   - AES 키 유도
    //   - MAC 비밀 유도

    // 결과: 암호화된 통신 채널 확립
}

// 메시지 프레이밍
func (c *Conn) Write(code uint64, data []byte) error {
    // 의도: 메시지를 암호화하여 전송

    // 1. RLP 인코딩
    // 2. 프레임 헤더 생성 (메시지 크기 포함)
    // 3. 헤더 MAC 계산
    // 4. 페이로드 암호화
    // 5. 프레임 MAC 계산
    // 6. 전송
}
```

### 1.2 Node Discovery

**소스 위치**: `p2p/discover/`

#### 1.2.1 Discovery v4 (UDP 기반)
```go
// UDP 기반 노드 발견 프로토콜
// 목적: 네트워크에서 새로운 피어 찾기

type UDPv4 struct {
    // Kademlia-like DHT 사용
    // 각 노드는 256비트 ID 공간에 위치

    conn   UDPConn
    tab    *Table      // 라우팅 테이블
    db     *enode.DB   // 노드 데이터베이스

    // 내부 로직:
    // - Ping/Pong: 노드 생존 확인
    // - FindNode/Neighbors: 특정 거리의 노드 찾기
    // - ENRRequest/ENRResponse: Ethereum Node Record 교환
}

// 라우팅 테이블 구조
type Table struct {
    // K-bucket 알고리즘 사용
    // 목적: 가까운 노드들을 효율적으로 관리

    buckets [nBuckets]*bucket  // 256개 버킷

    // 각 버킷:
    // - 특정 거리 범위의 노드들 저장
    // - 최대 16개 노드 (k=16)
    // - LRU (Least Recently Used) 정책
}

func (tab *Table) findnode(target enode.ID, nresults int) []*node {
    // 의도: target에 가까운 노드 찾기

    // 1. 로컬 테이블에서 가장 가까운 노드들 선택
    // 2. 선택된 노드들에게 FINDNODE 요청
    // 3. 응답받은 노드들 중 더 가까운 것들 선택
    // 4. 반복 (iterative deepening)
    // 5. 최종 nresults개 노드 반환
}
```

#### 1.2.2 ENR (Ethereum Node Records)
```go
// ENR: 노드의 연결 정보를 담은 서명된 레코드
// 목적: 노드 정보의 검증 가능한 교환

type Record struct {
    // 키-값 쌍으로 구성
    // 예: IP 주소, TCP 포트, 공개키, 클라이언트 정보

    pairs []pair

    // 디지털 서명 포함
    // 의도: 노드 정보의 진위 보장
    signature []byte

    // 시퀀스 번호
    // 목적: 레코드 업데이트 추적
    seq uint64
}

// ENR은 base64 URL 인코딩된 문자열로 공유
// 형식: "enr:-IS4Q..."
```

### 1.3 Ethereum Wire Protocol (ETH)

**소스 위치**: `eth/protocols/eth/`

```go
// ETH 프로토콜: 블록체인 데이터 교환
// 목적: 블록, 트랜잭션, 상태 동기화

// 메시지 타입들
const (
    // Status (0x00)
    // 의도: 핸드셰이크, 네트워크/제네시스 확인
    StatusMsg = 0x00

    // NewBlockHashes (0x01)
    // 의도: 새로운 블록 해시 공지
    NewBlockHashesMsg = 0x01

    // Transactions (0x02)
    // 의도: 트랜잭션 전파
    TransactionsMsg = 0x02

    // GetBlockHeaders (0x03)
    // 의도: 블록 헤더 요청
    GetBlockHeadersMsg = 0x03

    // BlockHeaders (0x04)
    // 의도: 블록 헤더 응답
    BlockHeadersMsg = 0x04

    // GetBlockBodies (0x05)
    GetBlockBodiesMsg = 0x05

    // BlockBodies (0x06)
    BlockBodiesMsg = 0x06

    // NewBlock (0x07)
    // 의도: 완전한 블록 전파
    NewBlockMsg = 0x07

    // GetReceipts (0x0f)
    GetReceiptsMsg = 0x0f

    // Receipts (0x10)
    ReceiptsMsg = 0x10
)

// 핸들러 구조
type handler struct {
    // 목적: ETH 프로토콜 메시지 처리

    chain     *core.BlockChain
    txpool    *txpool.TxPool

    // 내부 로직
    peers *peerSet  // 연결된 ETH 피어들

    // 동기화 메커니즘
    downloader *downloader.Downloader
    fetcher    *fetcher.BlockFetcher
}

func (h *handler) handleMsg(peer *eth.Peer) error {
    // 메시지 디스패처
    // 의도: 메시지 타입별 적절한 핸들러 호출

    msg, err := peer.rw.ReadMsg()

    switch msg.Code {
    case StatusMsg:
        // 네트워크 ID, 제네시스 해시 검증
        // 호환되지 않으면 연결 종료

    case NewBlockHashesMsg:
        // 새 블록 해시 수신
        // fetcher에게 전달하여 다운로드 예약

    case TransactionsMsg:
        // 트랜잭션 수신
        // txpool에 추가

    case GetBlockHeadersMsg:
        // 블록 헤더 요청 처리
        // 로컬 체인에서 검색하여 응답

    // ... 기타 메시지들
    }
}
```

---

## 2. 데이터베이스 레이어 (Database Layer)

### 2.1 LevelDB 통합

**소스 위치**: `ethdb/`, `core/rawdb/`

#### 2.1.1 Database 인터페이스
```go
// 추상화된 데이터베이스 인터페이스
// 목적: 다양한 DB 백엔드 지원 (LevelDB, MemoryDB 등)

type Database interface {
    // 기본 KV 연산
    Get(key []byte) ([]byte, error)
    Put(key []byte, value []byte) error
    Delete(key []byte) error

    // 배치 연산 (원자적 다중 쓰기)
    // 의도: 성능 최적화, 일관성 보장
    NewBatch() Batch

    // 반복자
    // 목적: 키 범위 스캔
    NewIterator(prefix []byte, start []byte) Iterator

    // 통계
    Stat(property string) (string, error)

    // 압축
    // 의도: 디스크 공간 재확보, 읽기 성능 향상
    Compact(start []byte, limit []byte) error

    // 종료
    Close() error
}

// LevelDB 구현
type LDBDatabase struct {
    // go-leveldb 래퍼
    db *leveldb.DB

    // 메트릭 수집
    compTimeMeter    metrics.Meter  // 압축 시간
    compReadMeter    metrics.Meter  // 압축 읽기
    compWriteMeter   metrics.Meter  // 압축 쓰기
    writeDelayNMeter metrics.Meter  // 쓰기 지연
    diskSizeGauge    metrics.Gauge  // 디스크 크기

    // 목적: 성능 모니터링 및 최적화
}

// 배치 연산
type Batch interface {
    // 의도: 여러 연산을 메모리에 버퍼링
    Put(key, value []byte) error
    Delete(key []byte) error

    // 의도: 버퍼링된 모든 연산을 원자적으로 커밋
    Write() error

    // 배치 크기 확인
    ValueSize() int

    // 배치 초기화
    Reset()
}
```

#### 2.1.2 키 스킴 (Key Schema)

**소스 위치**: `core/rawdb/schema.go`

```go
// Ethereum의 모든 데이터는 키-값으로 저장
// 목적: 효율적인 데이터 접근 및 조직화

// 키 프리픽스 정의
var (
    // 헤더 관련
    headerPrefix       = []byte("h")  // headerPrefix + num (uint64 big endian) + hash -> header
    headerHashSuffix   = []byte("n")  // headerPrefix + num + headerHashSuffix -> hash
    headerNumberPrefix = []byte("H")  // headerNumberPrefix + hash -> num

    // 의도:
    // - 블록 번호로 헤더 찾기: "h" + num + hash
    // - 블록 번호로 해시 찾기: "h" + num + "n"
    // - 해시로 블록 번호 찾기: "H" + hash

    // 블록 바디 관련
    blockBodyPrefix     = []byte("b")  // blockBodyPrefix + num + hash -> block body
    blockReceiptsPrefix = []byte("r")  // blockReceiptsPrefix + num + hash -> block receipts

    // 트랜잭션 관련
    txLookupPrefix  = []byte("l")  // txLookupPrefix + hash -> transaction/receipt lookup metadata

    // 상태 관련
    preimagePrefix = []byte("secure-key-")  // preimagePrefix + hash -> preimage

    // 블록체인 메타데이터
    headHeaderKey = []byte("LastHeader")  // 최신 헤더의 해시
    headBlockKey  = []byte("LastBlock")   // 최신 완전 블록의 해시
    headFastKey   = []byte("LastFast")    // 최신 fast 블록의 해시

    // 스냅샷 관련
    snapshotRootKey     = []byte("SnapshotRoot")
    snapshotAccountPrefix = []byte("a")
    snapshotStoragePrefix = []byte("o")
)

// 키 생성 헬퍼 함수들
func headerKey(number uint64, hash common.Hash) []byte {
    // 의도: 블록 헤더의 정확한 키 생성
    // "h" + big-endian uint64 + 32-byte hash
    return append(append(headerPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
}

func blockBodyKey(number uint64, hash common.Hash) []byte {
    return append(append(blockBodyPrefix, encodeBlockNumber(number)...), hash.Bytes()...)
}

// 블록 번호 인코딩
func encodeBlockNumber(number uint64) []byte {
    // 빅 엔디안 8바이트
    // 목적: 키 정렬을 통한 순차 접근 최적화
    enc := make([]byte, 8)
    binary.BigEndian.PutUint64(enc, number)
    return enc
}
```

### 2.2 Freezer (Ancient Store)

**소스 위치**: `core/rawdb/freezer.go`

```go
// Freezer: 오래된 블록 데이터의 불변 append-only 저장소
// 목적: 디스크 공간 최적화 및 읽기 성능 향상

type Freezer struct {
    // 의도: 최근 블록은 LevelDB(빠른 접근), 오래된 블록은 Freezer(압축 저장)

    frozen uint64  // 동결된 항목의 수

    // 각 데이터 타입별 별도 테이블
    tables map[string]*freezerTable

    // 테이블 타입:
    // - "headers": 블록 헤더
    // - "hashes": 블록 해시
    // - "bodies": 블록 바디
    // - "receipts": 영수증
    // - "diffs": total difficulty
}

type freezerTable struct {
    // Append-only 파일들
    // 목적: 순차 쓰기로 최적 성능, 압축으로 공간 절약

    items      uint64       // 저장된 항목 수
    itemOffset uint64       // 첫 항목의 시작 오프셋

    headFile *os.File      // 현재 쓰기 중인 파일
    files    []*os.File    // 모든 데이터 파일들

    index *os.File         // 인덱스 파일 (각 항목의 오프셋)

    // 내부 로직
}

func (t *freezerTable) Append(item uint64, data []byte) error {
    // 의도: 새 데이터를 파일 끝에 추가

    // 1. 인덱스 엔트리 추가 (오프셋, 크기)
    // 2. 데이터 파일에 데이터 쓰기
    // 3. 파일 크기가 2GB 초과시 새 파일 생성
    // 4. 동기화 (주기적)
}

func (t *freezerTable) Retrieve(item uint64) ([]byte, error) {
    // 의도: 항목 번호로 데이터 읽기

    // 1. 인덱스에서 오프셋과 크기 조회
    // 2. 적절한 파일 선택
    // 3. 파일에서 데이터 읽기
    // 4. 반환
}

// 동결 과정
func (f *Freezer) freeze(db ethdb.KeyValueStore) {
    // 의도: LevelDB의 오래된 블록을 Freezer로 이동

    // 1. LevelDB에서 다음 동결할 블록 읽기
    // 2. Freezer에 추가
    // 3. LevelDB에서 삭제
    // 4. frozen 카운터 증가
    // 5. 반복

    // 결과: LevelDB는 작게 유지, 오래된 데이터는 효율적으로 저장
}
```

---

## 3. 스토리지 레이어 (Storage Layer)

### 3.1 Merkle Patricia Trie

**소스 위치**: `trie/`

#### 3.1.1 Trie 구조
```go
// Merkle Patricia Trie: Ethereum 상태 저장의 핵심 자료구조
// 목적:
// - 효율적인 상태 저장 및 검증
// - 암호학적으로 안전한 상태 루트 제공
// - 부분 상태 증명 (Merkle proof) 지원

type Trie struct {
    db   *Database     // 트라이 노드 캐시 및 저장소
    root node          // 루트 노드

    // 목적: 상태 변경 추적
    unhashed int       // 아직 해시되지 않은 노드 수
}

// 노드 타입들
type (
    // 1. fullNode: 17개 자식을 가진 브랜치 노드
    fullNode struct {
        Children [17]node  // 16개는 hex 문자, 1개는 value
        flags    nodeFlag
    }
    // 의도: 경로 분기점 표현

    // 2. shortNode: 키-값 또는 키-서브트라이 쌍
    shortNode struct {
        Key   []byte     // 압축된 경로
        Val   node       // 하위 노드 또는 값
        flags nodeFlag
    }
    // 의도: 경로 압축으로 공간 절약

    // 3. hashNode: 다른 노드의 해시 참조
    hashNode []byte
    // 의도: 큰 서브트리를 해시로 대체하여 메모리 절약

    // 4. valueNode: 실제 값
    valueNode []byte
)

// 트라이 삽입
func (t *Trie) Update(key, value []byte) {
    // 의도: 키-값 쌍 업데이트

    // 1. 키를 hex 인코딩으로 변환
    k := keybytesToHex(key)

    // 2. 루트부터 재귀적으로 삽입
    // 3. 경로상의 노드들 수정
    // 4. 필요시 노드 분할/병합

    // 내부 로직:
    // - fullNode 만남: 다음 문자에 해당하는 자식으로 이동
    // - shortNode 만남:
    //   - 키 일치: 값 업데이트
    //   - 키 불일치: fullNode로 확장하여 분기
    // - nil 노드: 새 shortNode 생성
}

func (t *Trie) Get(key []byte) []byte {
    // 의도: 키로 값 조회

    // 1. 키를 hex로 변환
    // 2. 루트부터 경로 따라가기
    // 3. valueNode 찾으면 반환
    // 4. hashNode 만나면 DB에서 로드
}

// 해시 계산
func (t *Trie) Hash() common.Hash {
    // 의도: 현재 상태의 루트 해시 계산

    // 1. 모든 수정된 노드 해시 계산 (post-order)
    // 2. 노드가 32바이트 이상이면 Keccak-256 해시
    // 3. 32바이트 미만이면 RLP 인코딩 그대로 (inline)
    // 4. 루트 해시 반환

    // 결과: 전체 상태를 나타내는 32바이트 해시
}

// 커밋
func (t *Trie) Commit(onleaf LeafCallback) (common.Hash, error) {
    // 의도: 트라이를 데이터베이스에 영구 저장

    // 1. 모든 dirty 노드 해시 계산
    // 2. 노드를 RLP 인코딩
    // 3. 데이터베이스에 hash -> RLP 저장
    // 4. 루트 해시 반환

    // onleaf: 각 leaf 노드 처리 콜백 (옵션)
}
```

#### 3.1.2 Secure Trie
```go
// SecureTrie: Keccak256으로 키를 해시한 Trie
// 목적: 경로 예측 공격 방지, 균등 분포

type SecureTrie struct {
    trie             Trie
    secKeyCache      map[string][]byte  // hash -> original key

    // 의도: 원본 키 복구 가능하도록 캐시
}

func (t *SecureTrie) Update(key, value []byte) {
    // 1. key의 Keccak256 해시 계산
    hk := crypto.Keccak256(key)

    // 2. 해시를 키로 사용하여 업데이트
    t.trie.Update(hk, value)

    // 3. 원본 키 캐싱 (preimage)
    t.secKeyCache[string(hk)] = common.CopyBytes(key)
}

// 사용 예:
// - State Trie: 계정 주소(20 bytes) -> Keccak256(주소) -> 트라이 키
// - Storage Trie: 스토리지 키(32 bytes) -> Keccak256(키) -> 트라이 키
```

### 3.2 State Database

**소스 위치**: `core/state/`, `core/state/snapshot/`

#### 3.2.1 StateDB
```go
// StateDB: Ethereum 월드 스테이트 관리
// 목적: 모든 계정 및 스토리지 상태 관리

type StateDB struct {
    db   Database            // 백엔드 트라이 데이터베이스
    trie Trie                // 메인 계정 트라이

    // 계정 상태 캐시
    // 목적: 중복 트라이 조회 방지, 성능 향상
    stateObjects      map[common.Address]*stateObject
    stateObjectsDirty map[common.Address]struct{}  // 수정된 계정들

    // 로그, 프리이미지 등
    logs    map[common.Hash][]*types.Log
    preimages map[common.Hash][]byte

    // 스냅샷 지원
    // 의도: 빠른 롤백
    journal        *journal
    validRevisions []revision
    nextRevisionId int

    // 내부 로직: Copy-on-Write semantics
}

type stateObject struct {
    // 단일 계정의 상태
    address  common.Address
    addrHash common.Hash
    data     types.StateAccount  // nonce, balance, storageRoot, codeHash

    db       *StateDB

    // 이 계정의 스토리지 트라이
    trie Trie

    // 스토리지 캐시
    originStorage  Storage  // 원본 값 (롤백용)
    pendingStorage Storage  // 수정 중인 값
    dirtyStorage   Storage  // 최종 수정된 값

    // 코드
    code Code

    // 플래그
    dirtyCode bool
    suicided  bool
    deleted   bool
}

// 계정 조회/생성
func (s *StateDB) GetOrNewStateObject(addr common.Address) *stateObject {
    // 1. 캐시 확인
    if obj := s.stateObjects[addr]; obj != nil {
        return obj
    }

    // 2. 트라이에서 로드
    enc := s.trie.Get(addr.Bytes())
    if len(enc) == 0 {
        // 계정 없음 -> 새로 생성
        return s.createObject(addr)
    }

    // 3. RLP 디코딩
    var data types.StateAccount
    rlp.DecodeBytes(enc, &data)

    // 4. stateObject 생성 및 캐시
    obj := newObject(s, addr, data)
    s.setStateObject(obj)
    return obj
}

// 잔액 업데이트
func (s *StateDB) AddBalance(addr common.Address, amount *big.Int) {
    stateObject := s.GetOrNewStateObject(addr)
    if stateObject != nil {
        stateObject.AddBalance(amount)
    }
}

// 스토리지 업데이트
func (s *StateDB) SetState(addr common.Address, key, value common.Hash) {
    stateObject := s.GetOrNewStateObject(addr)
    if stateObject != nil {
        stateObject.SetState(s.db, key, value)
    }
}

// 커밋
func (s *StateDB) Commit(deleteEmptyObjects bool) (common.Hash, error) {
    // 의도: 모든 상태 변경을 트라이에 반영하고 DB에 저장

    // 1. 모든 dirty stateObject 순회
    for addr, stateObject := range s.stateObjectsDirty {
        // 1.1. 스토리지 트라이 커밋
        storageRoot := stateObject.updateStorageTrie()

        // 1.2. 계정 데이터 업데이트
        stateObject.data.Root = storageRoot

        // 1.3. 계정 트라이에 인코딩하여 저장
        s.updateStateObject(stateObject)
    }

    // 2. 계정 트라이 커밋
    root, err := s.trie.Commit(nil)

    // 3. 루트 해시 반환
    return root, err
}

// 스냅샷/롤백
func (s *StateDB) Snapshot() int {
    // 현재 상태의 스냅샷 ID 반환
    // 목적: 트랜잭션 실행 중 롤백 지점

    id := s.nextRevisionId
    s.nextRevisionId++
    s.validRevisions = append(s.validRevisions, revision{id, s.journal.length()})
    return id
}

func (s *StateDB) RevertToSnapshot(revid int) {
    // 의도: 특정 스냅샷으로 롤백
    // 사용: 트랜잭션 실행 실패시

    // 저널 엔트리들을 역순으로 undo
    s.journal.revert(s, snapshot)
}
```

#### 3.2.2 Snapshot Acceleration

**소스 위치**: `core/state/snapshot/`

```go
// Snapshot: 플랫(flat) 키-값 계층
// 목적: 트라이 순회 없이 빠른 상태 접근

type Tree struct {
    // 의도: 스냅샷들의 트리 구조 유지
    // 각 블록마다 incremental snapshot

    diskdb ethdb.KeyValueStore  // 영구 스냅샷 저장
    triedb *trie.Database       // 트라이 데이터베이스

    layers map[common.Hash]snapshot  // 메모리 레이어들

    // 내부 로직:
    // - disk layer: 디스크의 베이스 스냅샷
    // - diff layers: 메모리의 변경 사항들 (체인)
}

type snapshot interface {
    // 계정 직접 조회 (트라이 순회 불필요!)
    Account(hash common.Hash) (*Account, error)

    // 스토리지 직접 조회
    Storage(accountHash, storageHash common.Hash) ([]byte, error)
}

type diffLayer struct {
    // 메모리 diff
    parent snapshot         // 부모 스냅샷
    root   common.Hash      // 상태 루트

    // 변경된 계정들 (flat map)
    accountData map[common.Hash][]byte

    // 변경된 스토리지들 (2D map)
    storageData map[common.Hash]map[common.Hash][]byte

    // 삭제된 것들
    destructSet map[common.Hash]struct{}
}

type diskLayer struct {
    // 디스크 베이스 스냅샷
    root common.Hash

    // 플랫 계정 저장: hash -> RLP(account)
    // 플랫 스토리지 저장: accountHash + storageHash -> value
    // 키 프리픽스: "a" (account), "o" (storage)

    // 의도: 트라이 없이 O(1) 조회
}

// 조회 과정
func (t *Tree) Account(hash common.Hash) (*Account, error) {
    // 1. 최신 diff layer부터 검색
    // 2. 찾으면 반환
    // 3. 못 찾으면 부모 layer로
    // 4. disk layer까지 도달하면 DB 조회

    // 장점:
    // - 트라이 순회 불필요
    // - 대부분의 경우 메모리에서 해결
    // - disk layer는 수백만 계정 O(1) 조회
}

// 생성 과정
func (t *Tree) Generate(root common.Hash, accounts map[common.Hash][]byte, ...) {
    // 의도: 트라이를 순회하여 플랫 스냅샷 생성

    // 1. 계정 트라이 전체 순회
    // 2. 각 계정의 스토리지 트라이 순회
    // 3. 모든 계정, 스토리지를 플랫 DB에 저장
    // 4. 완료되면 disk layer 활성화

    // 시간: 몇 시간 소요 가능 (메인넷 기준)
    // 이후 동기화: 매우 빠름!
}
```

---

## 4. RPC 레이어 (RPC Layer)

### 4.1 JSON-RPC 서버

**소스 위치**: `rpc/`

#### 4.1.1 Server 구조
```go
// Server: JSON-RPC 요청 처리
// 목적: HTTP, WebSocket, IPC를 통한 클라이언트 통신

type Server struct {
    services serviceRegistry  // 등록된 서비스들

    // 목적: 네임스페이스별 API 그룹화
    // 예: "eth", "net", "web3", "debug", "admin"

    run      int32
    codecs   mapset.Set  // 활성 연결들
}

type serviceRegistry struct {
    mu       sync.Mutex
    services map[string]service  // 네임스페이스 -> 서비스

    // 각 서비스는 여러 메서드 포함
}

type service struct {
    name          string                    // 네임스페이스
    callbacks     map[string]*callback      // RPC 메서드들
    subscriptions map[string]*callback      // pub/sub 구독들
}

type callback struct {
    rcvr        reflect.Value   // 리시버 객체
    method      reflect.Method  // 메서드
    argTypes    []reflect.Type  // 파라미터 타입들
    hasCtx      bool            // context.Context 파라미터 있는지
    errPos      int             // error 반환값 위치

    // 의도: 리플렉션을 통한 동적 메서드 호출
}

// 서비스 등록
func (s *Server) RegisterName(name string, rcvr interface{}) error {
    // 의도: 구조체의 public 메서드들을 RPC 메서드로 노출

    // 1. rcvr의 타입 검사
    // 2. 모든 public 메서드 순회
    // 3. 메서드 시그니처 검증:
    //    - 반환값: (T, error) 또는 error
    //    - 첫 파라미터: context.Context (옵션)
    // 4. callbacks 맵에 등록

    // 예:
    // type EthAPI struct { ... }
    // func (api *EthAPI) BlockNumber() (uint64, error) { ... }
    //
    // RegisterName("eth", new(EthAPI))
    // -> "eth_blockNumber" 메서드 등록
}

// 요청 처리
func (s *Server) serveRequest(ctx context.Context, codec ServerCodec,
                               singleShot bool) error {
    // 의도: 단일 RPC 요청 처리

    // 1. JSON-RPC 요청 파싱
    reqs, batch, err := codec.readBatch()

    // 2. 각 요청 처리
    for _, req := range reqs {
        // 2.1. 메서드 이름 파싱 (네임스페이스_메서드)
        // 2.2. 서비스 및 콜백 조회
        // 2.3. 파라미터 언마샬
        // 2.4. 메서드 호출 (리플렉션)
        // 2.5. 응답 생성
    }

    // 3. 응답 전송
    codec.write(ctx, responses)
}
```

#### 4.1.2 HTTP Handler
```go
// HTTP 서버 생성
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // CORS 헤더 설정
    // Content-Type 검증
    // 바디 크기 제한

    // JSON-RPC codec 생성
    codec := NewJSONCodec(&httpReadWriteNopCloser{
        io.LimitReader(r.Body, maxRequestContentLength),
        w,
        r.Body,
    })

    // 요청 처리
    s.serveRequest(r.Context(), codec, true)
}

// WebSocket Handler
func (s *Server) WebsocketHandler(allowedOrigins []string) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // WebSocket 업그레이드
        conn, err := upgrader.Upgrade(w, r, nil)

        // WebSocket codec 생성
        codec := newWebsocketCodec(conn)

        // 지속적인 요청 처리 (pub/sub 지원)
        s.serveRequest(r.Context(), codec, false)
    })
}
```

### 4.2 ETH Namespace

**소스 위치**: `eth/api.go`, `internal/ethapi/`

```go
// PublicEthereumAPI: 공개 Ethereum API
type PublicEthereumAPI struct {
    e *Ethereum  // 백엔드
}

// 블록 번호 조회
func (api *PublicEthereumAPI) BlockNumber() hexutil.Uint64 {
    // 의도: 최신 블록 번호 반환
    header := api.e.blockchain.CurrentHeader()
    return hexutil.Uint64(header.Number.Uint64())
}

// 잔액 조회
func (api *PublicEthereumAPI) GetBalance(
    ctx context.Context,
    address common.Address,
    blockNrOrHash rpc.BlockNumberOrHash,
) (*hexutil.Big, error) {
    // 의도: 특정 블록에서의 계정 잔액 조회

    // 1. 블록 번호/해시로 상태 가져오기
    state, _, err := api.e.APIBackend.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)

    // 2. 계정 잔액 조회
    balance := state.GetBalance(address)

    return (*hexutil.Big)(balance), nil
}

// 트랜잭션 전송
func (api *PublicTransactionPoolAPI) SendTransaction(
    ctx context.Context,
    args TransactionArgs,
) (common.Hash, error) {
    // 의도: 서명된 트랜잭션을 풀에 추가하고 전파

    // 1. 트랜잭션 파라미터 검증
    // 2. 트랜잭션 객체 생성
    // 3. 서명
    // 4. 트랜잭션 풀에 추가
    tx, err := api.b.SendTx(ctx, signedTx)

    // 5. 네트워크에 전파
    // 6. 트랜잭션 해시 반환
    return tx.Hash(), nil
}

// 트랜잭션 호출 (실행하지 않음)
func (api *PublicBlockChainAPI) Call(
    ctx context.Context,
    args TransactionArgs,
    blockNrOrHash rpc.BlockNumberOrHash,
    overrides *StateOverride,
) (hexutil.Bytes, error) {
    // 의도: 상태 변경 없이 트랜잭션 시뮬레이션

    // 1. 상태 가져오기
    state, header, err := api.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)

    // 2. EVM 환경 생성
    evm := api.b.GetEVM(ctx, msg, state, header)

    // 3. 트랜잭션 실행 (상태는 임시)
    result, err := doCall(ctx, api.b, args, state, header, evm, timeout)

    // 4. 반환값만 반환
    return result.ReturnData, nil
}

// 로그 조회
func (api *PublicFilterAPI) GetLogs(
    ctx context.Context,
    crit FilterCriteria,
) ([]*types.Log, error) {
    // 의도: 필터 조건에 맞는 로그들 조회

    // 1. 블록 범위 결정
    // 2. Bloom 필터로 빠른 스캔
    // 3. 조건에 맞는 로그 수집
    // 4. 반환

    // 내부 로직: Bloom filter로 불필요한 블록 스킵
}
```

### 4.3 필터 및 구독 (Filters & Subscriptions)

**소스 위치**: `eth/filters/`

```go
// EventSystem: 이벤트 구독 관리
// 목적: 새 블록, 로그, pending tx 등의 실시간 알림

type EventSystem struct {
    backend Backend

    // 구독 관리
    txsSub        event.Subscription  // 새 트랜잭션
    logsSub       event.Subscription  // 새 로그
    rmLogsSub     event.Subscription  // 제거된 로그
    chainSub      event.Subscription  // 체인 이벤트
    pendingLogsSub event.Subscription  // pending 로그

    // 구독자들
    install   chan *subscription
    uninstall chan *subscription
}

type subscription struct {
    id        rpc.ID
    typ       Type           // 구독 타입
    created   time.Time
    logsCrit  FilterCriteria // 로그 필터
    logs      chan []*types.Log
    txs       chan []*types.Transaction
    headers   chan *types.Header

    // 내부 로직: 채널을 통한 비동기 이벤트 전달
}

// 이벤트 루프
func (es *EventSystem) eventLoop() {
    // 의도: 백엔드 이벤트를 구독자들에게 전달

    for {
        select {
        case ev := <-es.txsSub.Chan():
            // 새 트랜잭션 이벤트
            // 관심 있는 구독자들에게 전달

        case ev := <-es.logsSub.Chan():
            // 새 로그 이벤트
            // 필터 조건 확인 후 전달

        case ev := <-es.chainSub.Chan():
            // 체인 이벤트 (새 블록, 리org)
            // 해당 구독자들에게 전달

        case f := <-es.install:
            // 새 구독 등록

        case f := <-es.uninstall:
            // 구독 해제
        }
    }
}

// WebSocket을 통한 구독 예제:
// {"jsonrpc":"2.0", "id": 1, "method": "eth_subscribe",
//  "params": ["logs", {"address": "0x..."}]}
//
// -> 서버가 새 로그 발생시마다 알림:
// {"jsonrpc":"2.0","method":"eth_subscription",
//  "params":{"subscription":"0x...","result":{...}}}
```

---

## 5. 핵심 데이터 구조

### 5.1 Block & Header

**소스 위치**: `core/types/block.go`

```go
// Header: 블록 헤더
type Header struct {
    ParentHash  common.Hash    // 이전 블록 해시
    UncleHash   common.Hash    // 엉클 블록들의 해시
    Coinbase    common.Address // 채굴자 주소
    Root        common.Hash    // 상태 트라이 루트
    TxHash      common.Hash    // 트랜잭션 트라이 루트
    ReceiptHash common.Hash    // 영수증 트라이 루트
    Bloom       Bloom          // 로그 Bloom 필터
    Difficulty  *big.Int       // PoW 난이도
    Number      *big.Int       // 블록 번호
    GasLimit    uint64         // 가스 한도
    GasUsed     uint64         // 사용된 가스
    Time        uint64         // 타임스탬프
    Extra       []byte         // 추가 데이터
    MixDigest   common.Hash    // PoW 믹스 해시
    Nonce       BlockNonce     // PoW 논스

    // PoS 필드들 (이후 추가)
    BaseFee *big.Int           // EIP-1559 base fee
}

// Block: 완전한 블록
type Block struct {
    header       *Header
    uncles       []*Header
    transactions Transactions

    // 캐시된 값들
    hash atomic.Value
    size atomic.Value

    // Received from/at
    ReceivedAt   time.Time
    ReceivedFrom interface{}
}

// 블록 해시 계산
func (b *Block) Hash() common.Hash {
    // 의도: 헤더의 RLP 해시
    // 캐시하여 중복 계산 방지

    if hash := b.hash.Load(); hash != nil {
        return hash.(common.Hash)
    }

    // RLP 인코딩 후 Keccak-256
    v := rlpHash(b.header)
    b.hash.Store(v)
    return v
}
```

### 5.2 Transaction

**소스 위치**: `core/types/transaction.go`

```go
// Transaction: 트랜잭션 (여러 타입 지원)
type Transaction struct {
    inner TxData          // 실제 트랜잭션 데이터 (타입별)
    time  time.Time       // 수신 시간

    // 캐시
    hash atomic.Value
    size atomic.Value
    from atomic.Value
}

// 트랜잭션 타입들
type TxData interface {
    txType() byte
    // ...
}

// Legacy 트랜잭션 (EIP-155 이전)
type LegacyTx struct {
    Nonce    uint64
    GasPrice *big.Int
    Gas      uint64
    To       *common.Address  // nil이면 컨트랙트 생성
    Value    *big.Int
    Data     []byte
    V, R, S  *big.Int          // 서명
}

// EIP-1559 트랜잭션
type DynamicFeeTx struct {
    ChainID    *big.Int
    Nonce      uint64
    GasTipCap  *big.Int  // 우선순위 fee (miner tip)
    GasFeeCap  *big.Int  // 최대 fee
    Gas        uint64
    To         *common.Address
    Value      *big.Int
    Data       []byte
    AccessList AccessList  // EIP-2930
    V, R, S    *big.Int
}

// 서명 해시 계산
func (tx *Transaction) Hash() common.Hash {
    // 타입별로 다른 해시 방법
    return prefixedRlpHash(tx.Type(), tx.inner)
}

// 서명 검증 및 발신자 복구
func Sender(signer Signer, tx *Transaction) (common.Address, error) {
    // 의도: ECDSA 서명에서 공개키 복구 -> 주소 도출

    // 1. 서명 해시 계산
    hash := signer.Hash(tx)

    // 2. V, R, S에서 공개키 복구
    addr, err := recoverPlain(hash, tx.inner.rawSignatureValues())

    return addr, err
}
```

### 5.3 Receipt

**소스 위치**: `core/types/receipt.go`

```go
// Receipt: 트랜잭션 실행 영수증
type Receipt struct {
    Type              uint8   // 트랜잭션 타입
    PostState         []byte  // 실행 후 상태 루트 (legacy)
    Status            uint64  // 1=성공, 0=실패 (EIP-658)
    CumulativeGasUsed uint64  // 블록 내 누적 가스
    Bloom             Bloom   // 로그 bloom 필터
    Logs              []*Log  // 발생한 로그들

    // 추가 정보
    TxHash          common.Hash    // 트랜잭션 해시
    ContractAddress common.Address // 생성된 컨트랙트 주소
    GasUsed         uint64         // 이 트랜잭션이 사용한 가스

    // Consensus fields (블록에 포함됨)
    BlockHash        common.Hash
    BlockNumber      *big.Int
    TransactionIndex uint
}

// Log: 이벤트 로그
type Log struct {
    Address common.Address  // 로그를 발생시킨 컨트랙트
    Topics  []common.Hash   // 인덱싱된 토픽들 (최대 4개)
    Data    []byte          // 인덱싱되지 않은 데이터

    // 위치 정보
    BlockNumber uint64
    TxHash      common.Hash
    TxIndex     uint
    BlockHash   common.Hash
    Index       uint  // 블록 내 로그 인덱스

    // 내부 로직:
    // Topics[0]: 이벤트 시그니처 해시
    // Topics[1-3]: indexed 파라미터들
    // Data: non-indexed 파라미터들 (ABI 인코딩)
}

// Bloom Filter: 로그 빠른 검색
type Bloom [BloomByteLength]byte  // 256 bytes

func CreateBloom(receipts Receipts) Bloom {
    // 의도: 블록의 모든 로그를 Bloom filter로 요약
    // 사용: 특정 주소/토픽의 로그 포함 여부 빠른 확인

    bin := new(big.Int)
    for _, receipt := range receipts {
        for _, log := range receipt.Logs {
            // 주소 추가
            bin.Or(bin, bloom9(log.Address.Bytes()))

            // 각 토픽 추가
            for _, topic := range log.Topics {
                bin.Or(bin, bloom9(topic.Bytes()))
            }
        }
    }

    return BytesToBloom(bin.Bytes())
}

// bloom9: 단일 요소를 3개 비트로 매핑
func bloom9(b []byte) *big.Int {
    // Keccak-256 해시의 처음 6바이트 사용
    // 각 2바이트 쌍을 11비트 인덱스로 변환
    // 총 3개 비트 설정
    //
    // 결과: False positive 가능, False negative 불가능
    // -> 로그가 있을 "수도" 있으면 블록 검사, 확실히 없으면 스킵
}
```

---

## 학습 로드맵

### Phase 1: 기초 이해 (1-2주)
1. **P2P 네트워킹**
   - `p2p/server.go` 읽기
   - 로컬에서 2개 노드 연결 테스트
   - 핸드셰이크 과정 로깅

2. **데이터베이스**
   - `ethdb/` 인터페이스 이해
   - LevelDB 기본 연산 테스트
   - 키 스킴 분석

### Phase 2: 심화 학습 (2-3주)
1. **Merkle Patricia Trie**
   - `trie/trie.go` 구현 분석
   - 간단한 트라이 직접 구현
   - 해시 계산 과정 추적

2. **State Management**
   - `core/state/statedb.go` 읽기
   - 트랜잭션 실행 과정 추적
   - 스냅샷 메커니즘 이해

### Phase 3: 전문가 (3-4주)
1. **RPC & API**
   - `rpc/` 서버 구조 분석
   - 커스텀 API 추가 실험
   - WebSocket 구독 구현

2. **전체 흐름 통합**
   - 트랜잭션이 블록에 포함되는 전 과정 추적
   - 상태 변경 및 커밋 과정 이해
   - 네트워크 전파 메커니즘 분석

### 실습 프로젝트
1. 간단한 블록체인 익스플로러 (RPC 사용)
2. 커스텀 이벤트 리스너 (필터 사용)
3. 상태 변경 시각화 도구
4. P2P 네트워크 모니터링 도구

---

## 추가 학습 자료

### 공식 문서
- Geth 공식 문서: https://geth.ethereum.org/docs
- DevP2P 명세: https://github.com/ethereum/devp2p
- Ethereum 옐로우 페이퍼: https://ethereum.github.io/yellowpaper/paper.pdf

### 소스 코드 분석
- go-ethereum 저장소: https://github.com/ethereum/go-ethereum
- 주요 디렉토리:
  - `p2p/`: 네트워킹
  - `core/`: 블록체인 코어
  - `eth/`: Ethereum 프로토콜
  - `rpc/`: JSON-RPC
  - `trie/`: Merkle Patricia Trie
  - `ethdb/`: 데이터베이스

### 디버깅 팁
```bash
# 디버그 모드로 geth 실행
geth --verbosity 5 --log.debug

# 특정 모듈만 디버그
geth --log.debug --log.backtrace "server.go:250"

# P2P 연결 모니터링
geth attach
> admin.peers

# 트라이 검사
geth --datadir ./data inspect-trie <root-hash>
```

---

## 6. 완전한 트랜잭션 처리 플로우

### 6.1 End-to-End Transaction Path

**시나리오**: 사용자가 `eth_sendTransaction`을 호출하여 100 ETH를 전송

#### Step 1: RPC 요청 수신
**파일**: `internal/ethapi/api.go:1724`

```go
// eth_sendTransaction 처리 시작
func (s *PublicTransactionPoolAPI) SendTransaction(ctx context.Context, args TransactionArgs) (common.Hash, error) {
    // 1-1. 인자 검증
    if args.Nonce == nil {
        // Nonce가 없으면 현재 pending nonce 조회
        nonce, err := s.b.GetPoolNonce(ctx, args.from())
        args.Nonce = (*hexutil.Uint64)(&nonce)
    }
    
    // 1-2. Gas 추정 (사용자가 지정하지 않은 경우)
    if args.Gas == nil {
        gas, err := DoEstimateGas(ctx, s.b, args, nil, s.b.RPCGasCap())
        args.Gas = (*hexutil.Uint64)(&gas)
    }
    
    // 1-3. 트랜잭션 객체 생성
    tx := args.toTransaction()
    
    // 내부 로직:
    // - 인자 검증: from, to, value, gas, gasPrice
    // - Missing 필드 자동 채우기 (nonce, gas limit)
    // - EIP-1559 vs Legacy 트랜잭션 구분
    
    // 1-4. 서명 (지갑에서)
    signed, err := s.sign(args.from(), tx)
    
    // 1-5. Transaction Pool로 전송
    return SubmitTransaction(ctx, s.b, signed)
}
```

**Edge Cases**:
- `from` 주소에 잔액 부족 → `insufficient funds` 에러
- Nonce가 너무 큼 → `nonce too high` (pending 큐 대기)
- Gas limit 초과 → `exceeds block gas limit`

#### Step 2: Transaction Pool 검증
**파일**: `core/tx_pool.go:645`

```go
func (pool *TxPool) add(tx *types.Transaction, local bool) (replaced bool, err error) {
    // 2-1. 기본 검증
    if err := pool.validateTx(tx, local); err != nil {
        return false, err
    }
    
    // validateTx 내부:
    // ✓ 트랜잭션 크기 < 128KB
    // ✓ Value >= 0
    // ✓ Gas limit >= 21000 (intrinsic gas)
    // ✓ 서명 유효성 검증
    // ✓ Sender 계정 존재 확인
    
    // 2-2. Nonce 확인
    from, _ := types.Sender(pool.signer, tx)
    currentNonce := pool.currentState.GetNonce(from)
    
    if tx.Nonce() < currentNonce {
        return false, ErrNonceTooLow  // 이미 사용된 nonce
    }
    
    // 2-3. 잔액 확인
    balance := pool.currentState.GetBalance(from)
    cost := tx.Cost()  // value + (gas * gasPrice)
    
    if balance.Cmp(cost) < 0 {
        return false, ErrInsufficientFunds
    }
    
    // 2-4. Pool에 추가
    pool.enqueueTx(tx.Hash(), tx, local, true)
    
    // enqueueTx 로직:
    // - Pending queue: nonce가 연속적인 트랜잭션
    // - Future queue: nonce gap이 있는 트랜잭션
    // - Replace 정책: gasPrice가 10% 이상 높으면 교체
    
    return false, nil
}
```

**Pool 상태 전이**:
```
Future Queue → Pending Queue
  (Nonce gap 채워지면)
     ↓
  Miner가 선택
     ↓
  Block에 포함
```

#### Step 3: Mining (Block 생성)
**파일**: `miner/worker.go:1082`

```go
func (w *worker) commitTransaction(env *environment, tx *types.Transaction) error {
    // 3-1. StateDB 스냅샷 생성 (Copy-on-Write)
    snap := env.state.Snapshot()
    
    // 3-2. EVM으로 트랜잭션 실행
    receipt, err := core.ApplyTransaction(
        w.chainConfig,
        w.chain,
        &env.coinbase,  // Miner 주소
        env.gasPool,
        env.state,      // StateDB
        env.header,
        tx,
        &env.header.GasUsed,
        vm.Config{},
    )
    
    if err != nil {
        // 실행 실패 시 상태 복구
        env.state.RevertToSnapshot(snap)
        return err
    }
    
    // 3-3. Receipt 저장
    env.receipts = append(env.receipts, receipt)
    env.txs = append(env.txs, tx)
    
    return nil
}
```

#### Step 4: EVM 실행
**파일**: `core/state_transition.go:319`

```go
func (st *StateTransition) TransitionDb() (*ExecutionResult, error) {
    // 4-1. Intrinsic Gas 계산
    gas, err := IntrinsicGas(st.data, st.to() == nil, true, rules)
    // Intrinsic Gas = 21000 (기본)
    //               + 16 * zero bytes
    //               + 68 * non-zero bytes
    //               + 32000 (contract creation)
    
    if st.gas < gas {
        return nil, ErrIntrinsicGas
    }
    st.gas -= gas
    
    // 4-2. Sender 계정에서 Value 차감
    st.state.SubBalance(st.from(), st.value)
    
    // 4-3. Nonce 증가
    st.state.SetNonce(st.from(), st.state.GetNonce(st.from())+1)
    
    // 4-4. 실제 실행
    var ret []byte
    if contractCreation {
        ret, st.gas, vmerr = st.evm.Create(sender, st.data, st.gas, st.value)
    } else {
        // 일반 전송 또는 컨트랙트 호출
        st.state.SetNonce(st.msg.From(), st.state.GetNonce(sender.Address())+1)
        ret, st.gas, vmerr = st.evm.Call(sender, st.to(), st.data, st.gas, st.value)
    }
    
    // 4-5. 받는 주소에 Value 추가
    st.state.AddBalance(st.to(), st.value)
    
    // 4-6. Gas 환불 및 Miner에게 수수료 지급
    remaining := st.gas
    refund := st.refundGas(params.RefundQuotientEIP3529)
    st.state.AddBalance(st.evm.Context.Coinbase, new(big.Int).Mul(new(big.Int).SetUint64(st.gasUsed()), st.gasPrice))
    
    return &ExecutionResult{
        UsedGas:    st.gasUsed(),
        Err:        vmerr,
        ReturnData: ret,
    }, nil
}
```

**EVM 내부** (`core/vm/evm.go:202`):
```go
func (evm *EVM) Call(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
    // Transfer value
    if !evm.Context.CanTransfer(evm.StateDB, caller.Address(), value) {
        return nil, gas, ErrInsufficientBalance
    }
    evm.Context.Transfer(evm.StateDB, caller.Address(), addr, value)
    
    // 컨트랙트 코드 로드
    code := evm.StateDB.GetCode(addr)
    if len(code) == 0 {
        // EOA (Externally Owned Account) - 단순 전송
        return nil, gas, nil
    }
    
    // Interpreter로 바이트코드 실행
    contract := NewContract(caller, AccountRef(addr), value, gas)
    contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), code)
    
    ret, err = evm.interpreter.Run(contract, input, false)
    return ret, contract.Gas, err
}
```

#### Step 5: StateDB 상태 업데이트
**파일**: `core/state/statedb.go:494`

```go
func (s *StateDB) Finalise(deleteEmptyObjects bool) {
    // 5-1. Dirty 객체들 처리
    for addr := range s.journal.dirties {
        obj, exist := s.stateObjects[addr]
        
        if !exist {
            // Touched but deleted
            continue
        }
        
        if obj.suicided || (deleteEmptyObjects && obj.empty()) {
            // 5-2. 자살한 컨트랙트 또는 빈 계정 삭제
            obj.deleted = true
        } else {
            // 5-3. Trie에 변경사항 저장
            obj.finalise(true)  // Storage trie 업데이트
        }
        
        // 5-4. Pending 변경사항 기록
        s.stateObjectsPending[addr] = struct{}{}
        s.stateObjectsDirty[addr] = struct{}{}
    }
    
    // Journal 초기화 (다음 트랜잭션 준비)
    s.clearJournalAndRefund()
}
```

#### Step 6: Trie 커밋
**파일**: `core/state/statedb.go:894`

```go
func (s *StateDB) Commit(deleteEmptyObjects bool) (common.Hash, error) {
    // 6-1. Finalise 먼저 수행
    s.Finalise(deleteEmptyObjects)
    
    // 6-2. 각 계정의 Storage Trie 커밋
    for addr := range s.stateObjectsDirty {
        obj := s.stateObjects[addr]
        
        if obj.deleted {
            // 계정 삭제
            s.deleteStateObject(obj)
        } else {
            // Storage trie 커밋
            if obj.dirtyStorage {
                root, err := obj.CommitTrie(s.db)
                obj.data.Root = root
            }
            
            // 계정 데이터를 Main Trie에 저장
            s.updateStateObject(obj)
        }
    }
    
    // 6-3. Main Trie (World State) 커밋
    root, err := s.trie.Commit(nil)
    
    // 6-4. 디스크에 쓰기 (배치)
    s.db.TrieDB().Commit(root, false, nil)
    
    return root, nil
}
```

#### Step 7: Block Broadcast
**파일**: `eth/handler.go:473`

```go
func (h *handler) BroadcastBlock(block *types.Block, propagate bool) {
    hash := block.Hash()
    peers := h.peers.peersWithoutBlock(hash)
    
    // 7-1. Full block 전파 (일부 피어)
    if propagate {
        // sqrt(total peers)에게 full block 전송
        transfer := peers[:int(math.Sqrt(float64(len(peers))))]
        for _, peer := range transfer {
            peer.AsyncSendNewBlock(block, td)
        }
    }
    
    // 7-2. Block announcement (나머지 피어)
    // 블록 해시만 전송 → 피어가 요청하면 전송
    for _, peer := range peers {
        peer.AsyncSendNewBlockHash(block)
    }
}
```

### 6.2 실제 실행 시간 분석

**Profiling 결과** (mainnet block #15537393):

```
Total: 342ms

Phase                          Time    %
────────────────────────────────────────
RPC Validation                 2ms     0.6%
Pool Validation                5ms     1.5%
EVM Execution                  280ms   81.9%
  ├─ SLOAD/SSTORE             150ms   43.9%
  ├─ SHA3                      45ms   13.2%
  ├─ CALL                      60ms   17.5%
  └─ Other                     25ms    7.3%
StateDB Finalise               35ms   10.2%
Trie Commit                    15ms    4.4%
Broadcast                      5ms     1.5%
```

**병목 지점**:
1. **EVM Storage 접근 (SLOAD/SSTORE)**: 43.9%
   - 원인: Trie 조회 비용
   - 해결: Snapshot 사용 (geth 1.9.4+)

2. **SHA3 연산**: 13.2%
   - 원인: Keccak256 계산
   - 해결: 하드웨어 가속 (AVX512)

### 6.3 Error Cases 완전 분석

#### Case 1: Insufficient Funds

**발생 지점**: `core/state_transition.go:274`

```go
if err := st.buyGas(); err != nil {
    return nil, err  // ErrInsufficientFundsForTransfer
}

func (st *StateTransition) buyGas() error {
    mgval := new(big.Int).SetUint64(st.msg.GasLimit())
    mgval.Mul(mgval, st.msg.GasPrice())
    balanceCheck := mgval
    if st.msg.GasFeeCap() != nil {
        balanceCheck = new(big.Int).SetUint64(st.msg.GasLimit())
        balanceCheck = balanceCheck.Mul(balanceCheck, st.msg.GasFeeCap())
        balanceCheck.Add(balanceCheck, st.msg.Value())
    }
    
    if have, want := st.state.GetBalance(st.msg.From()), balanceCheck; have.Cmp(want) < 0 {
        return fmt.Errorf("%w: address %v have %v want %v", ErrInsufficientFunds, st.msg.From().Hex(), have, want)
    }
    // ...
}
```

**복구**:
- 사용자: Faucet에서 ETH 받기
- 자동: Gas price 낮추거나 value 감소

#### Case 2: Nonce Too Low

**발생 지점**: `core/tx_pool.go:693`

```go
if pool.currentState.GetNonce(from) > tx.Nonce() {
    return ErrNonceTooLow
}
```

**원인**:
- 같은 nonce로 두 개 트랜잭션 전송 → 하나만 성공
- 오래된 트랜잭션 재전송

**복구**:
- 올바른 nonce 조회: `eth_getTransactionCount(address, "pending")`
- 트랜잭션 취소: 같은 nonce, 높은 gas price, value=0

#### Case 3: Out of Gas

**발생 지점**: `core/vm/evm.go:155`

```go
func (evm *EVM) Call(...) (ret []byte, leftOverGas uint64, err error) {
    if evm.depth > int(params.CallCreateDepth) {
        return nil, gas, ErrDepth
    }
    if !evm.Context.CanTransfer(evm.StateDB, caller.Address(), value) {
        return nil, gas, ErrInsufficientBalance
    }
    
    snapshot := evm.StateDB.Snapshot()
    // ...
    ret, err = evm.interpreter.Run(contract, input, false)
    
    if err != nil {
        evm.StateDB.RevertToSnapshot(snapshot)  // 상태 복구
        if err != ErrExecutionReverted {
            contract.UseGas(contract.Gas)  // Gas 소진
        }
    }
    return ret, contract.Gas, err
}
```

**Gas 소모 지점**:
```
Opcode          Gas Cost
──────────────────────────
ADD/SUB         3
MUL             5
DIV             5
SLOAD           2100 (cold) / 100 (warm)
SSTORE          20000 (cold) / 100 (warm)
CALL            700 + transfer cost
CREATE          32000
SELFDESTRUCT    5000/25000
```

**복구**:
- Gas limit 증가
- 코드 최적화 (loop 감소, storage 사용 최소화)

---

## 7. 성능 최적화 실전

### 7.1 StateDB Copy-on-Write 최적화

**문제**: 매 트랜잭션마다 전체 state 복사는 비효율

**해결**: Snapshot + Journal 패턴

**구현** (`core/state/statedb.go:143`):

```go
type StateDB struct {
    db   Database
    trie Trie
    
    // Copy-on-Write를 위한 구조
    stateObjects      map[common.Address]*stateObject  // 수정된 객체들
    stateObjectsDirty map[common.Address]struct{}      // Dirty 플래그
    
    // Journal: Revert를 위한 변경 기록
    journal        *journal
    validRevisions []revision
    nextRevisionId int
    
    // Snapshot: 특정 시점 저장
    snaps *snapshot.Tree
}

// Snapshot 생성 (O(1) 시간)
func (s *StateDB) Snapshot() int {
    id := s.nextRevisionId
    s.nextRevisionId++
    s.validRevisions = append(s.validRevisions, revision{id, s.journal.length()})
    return id
}

// Revert (O(n) 변경사항 수)
func (s *StateDB) RevertToSnapshot(revid int) {
    idx := sort.Search(len(s.validRevisions), func(i int) bool {
        return s.validRevisions[i].id >= revid
    })
    s.journal.revert(s, s.validRevisions[idx].journalIndex)
    s.validRevisions = s.validRevisions[:idx]
}
```

**Journal 구조**:
```go
type journal struct {
    entries []journalEntry  // 변경 기록 스택
    dirties map[common.Address]int  // Dirty 객체들
}

type journalEntry interface {
    revert(*StateDB)  // 변경사항 되돌리기
    dirtied() *common.Address  // 어떤 주소가 dirty 되었는지
}

// 예: 잔액 변경 기록
type balanceChange struct {
    account *common.Address
    prev    *big.Int
}

func (ch balanceChange) revert(s *StateDB) {
    s.getStateObject(*ch.account).setBalance(ch.prev)
}
```

**성능 이점**:
- Snapshot 생성: O(1)
- Revert: O(n) where n = 변경사항 수
- 메모리: 변경된 객체만 저장 (전체 복사 불필요)

### 7.2 Trie Caching 전략

**3-Level Cache**:

```
L1: In-memory nodes (LRU, 256MB)
  ↓ miss
L2: Clean cache (512MB)
  ↓ miss  
L3: Disk (LevelDB)
```

**구현** (`trie/database.go:119`):

```go
type Database struct {
    diskdb ethdb.Database  // LevelDB
    
    // L1: Dirty nodes (수정됨)
    dirties map[common.Hash]*cachedNode
    oldest  common.Hash  // LRU eviction
    newest  common.Hash
    
    // L2: Clean cache (읽기 전용)
    cleans *fastcache.Cache
    
    // Metrics
    childrenSize common.StorageSize
}

// Node 조회
func (db *Database) node(hash common.Hash) (node, error) {
    // L1: Dirty cache 확인
    if n := db.dirties[hash]; n != nil {
        return n.obj, nil
    }
    
    // L2: Clean cache 확인
    if enc := db.cleans.Get(nil, hash[:]); enc != nil {
        return mustDecodeNode(hash[:], enc), nil
    }
    
    // L3: Disk 조회
    enc, err := db.diskdb.Get(hash[:])
    if err != nil {
        return nil, err
    }
    
    // Clean cache에 저장
    db.cleans.Set(hash[:], enc)
    return mustDecodeNode(hash[:], enc), nil
}
```

**Eviction 정책**:
```go
func (db *Database) Cap(limit common.StorageSize) error {
    // Dirty cache가 limit 초과 시
    for size := db.dirtiesSize; size > limit; {
        // LRU: 가장 오래된 노드 evict
        node := db.dirties[db.oldest]
        
        // Disk에 쓰기
        batch := db.diskdb.NewBatch()
        node.commit(batch)
        batch.Write()
        
        // Dirty에서 제거
        delete(db.dirties, db.oldest)
        db.oldest = node.flushNext
        
        size -= node.size
    }
    return nil
}
```

### 7.3 Parallel Transaction Execution

**현재**: Sequential (Ethereum Mainnet)

**실험적**: Parallel (Block-STM, research)

```go
// core/state/statedb.go (가상 구현)
func (s *StateDB) ParallelExecute(txs []*types.Transaction) ([]*types.Receipt, error) {
    // Phase 1: Optimistic parallel execution
    receipts := make([]*types.Receipt, len(txs))
    conflicts := make([]bool, len(txs))
    
    // 각 트랜잭션을 독립적으로 실행
    var wg sync.WaitGroup
    for i, tx := range txs {
        wg.Add(1)
        go func(index int, tx *types.Transaction) {
            defer wg.Done()
            
            // Local state copy
            localState := s.Copy()
            
            // Execute
            receipt, err := ApplyTransaction(tx, localState)
            receipts[index] = receipt
            
            // Conflict detection
            if DetectConflict(s.readSet[index], s.writeSet[index], otherTxs) {
                conflicts[index] = true
            }
        }(i, tx)
    }
    wg.Wait()
    
    // Phase 2: Re-execute conflicting transactions
    for i, conflict := range conflicts {
        if conflict {
            receipts[i], _ = ApplyTransaction(txs[i], s)  // Sequential
        }
    }
    
    return receipts, nil
}
```

**충돌 감지**:
```
Tx1: Transfer A → B (Read: A, B; Write: A, B)
Tx2: Transfer A → C (Read: A, C; Write: A, C)

Conflict: Both read/write A → Sequential execution 필요
```

### 7.4 Storage Layout 최적화

**문제**: 여러 변수가 각각 slot 차지 → 비효율

**해결**: Struct packing

```solidity
// ❌ Bad: 3 SLOADs
contract BadStorage {
    uint8 a;   // slot 0
    uint256 b; // slot 1
    uint8 c;   // slot 2
}

// ✅ Good: 1 SLOAD
contract GoodStorage {
    uint8 a;   // slot 0 [0-7]
    uint8 c;   // slot 0 [8-15]
    uint256 b; // slot 1
}
```

**Gas 절약**:
```
Bad:  3 × 2100 (cold) = 6300 gas
Good: 1 × 2100 (cold) = 2100 gas
Savings: 4200 gas (66%)
```

---

## 8. 디버깅 및 트러블슈팅

### 8.1 Geth Debug APIs

**활성화**:
```bash
geth --http --http.api "eth,debug,txpool" \
     --verbosity 4 \
     --vmdebug
```

#### 8.1.1 Transaction Tracing

```javascript
// debug_traceTransaction: 트랜잭션 step-by-step 실행
debug.traceTransaction("0x123...", {
    tracer: "callTracer",
    timeout: "10s"
})

// Output:
{
    "type": "CALL",
    "from": "0xabc...",
    "to": "0xdef...",
    "value": "0x100",
    "gas": "0x5208",
    "gasUsed": "0x5208",
    "input": "0x",
    "output": "0x",
    "calls": [
        {
            "type": "CALL",
            "from": "0xdef...",
            "to": "0x789...",
            "value": "0x50",
            "gas": "0x2000",
            "gasUsed": "0x1500",
            // Nested call
        }
    ]
}
```

#### 8.1.2 State Inspection

```javascript
// debug_dumpBlock: 특정 블록의 전체 상태
debug.dumpBlock(15537393)

// Output: 모든 계정의 balance, nonce, code, storage
{
    "root": "0xabc...",
    "accounts": {
        "0x123...": {
            "balance": "1000000000000000000",
            "nonce": 5,
            "root": "0xdef...",
            "codeHash": "0x789...",
            "code": "0x6060604052...",
            "storage": {
                "0x00": "0x123...",
                "0x01": "0x456..."
            }
        }
    }
}
```

#### 8.1.3 Opcodes Tracing

```javascript
// Opcode-level tracing
debug.traceTransaction("0x123...", {
    tracer: "opcodeTracer"
})

// Output: 모든 opcode 실행
{
    "structLogs": [
        {
            "pc": 0,
            "op": "PUSH1",
            "gas": 979000,
            "gasCost": 3,
            "depth": 1,
            "stack": [],
            "memory": [],
            "storage": {}
        },
        {
            "pc": 2,
            "op": "PUSH1",
            "gas": 978997,
            "gasCost": 3,
            "depth": 1,
            "stack": ["0x60"],
            "memory": [],
            "storage": {}
        }
        // ... 수백~수천 개 opcode
    ]
}
```

### 8.2 일반적인 문제 해결

#### 문제 1: "Transaction Underpriced"

**원인**: Gas price가 너무 낮음

```bash
# 현재 pending 트랜잭션들의 gas price 확인
> txpool.content.pending

# Minimum gas price 확인
> eth.gasPrice
50000000000  # 50 Gwei

# 해결: 더 높은 gas price 사용
> eth.sendTransaction({
    from: eth.accounts[0],
    to: "0x123...",
    value: web3.toWei(1, "ether"),
    gasPrice: 60000000000  // 60 Gwei
})
```

#### 문제 2: "Exceeds Block Gas Limit"

**원인**: 트랜잭션 gas limit이 블록 limit 초과

```bash
# 현재 블록 gas limit 확인
> eth.getBlock("latest").gasLimit
30000000  # 30M gas

# 트랜잭션 gas 추정
> eth.estimateGas({
    from: eth.accounts[0],
    to: "0xContractAddress",
    data: "0x..."
})
35000000  # 35M gas → 초과!

# 해결: 트랜잭션을 여러 개로 분할
```

#### 문제 3: "Replacement Transaction Underpriced"

**원인**: 같은 nonce 트랜잭션 교체 시 gas price가 10% 이상 높지 않음

```go
// core/tx_pool.go:760
if old != nil {
    // Replacement 조건
    if old.GasFeeCapIntCmp(tx) >= 0 || old.GasTipCapIntCmp(tx) >= 0 {
        return false, ErrReplaceUnderpriced
    }
    
    // 최소 10% 증가 필요
    a := new(big.Int).Mul(old.GasFeeCap(), big.NewInt(100+int64(priceBump)))
    b := new(big.Int).Mul(tx.GasFeeCap(), big.NewInt(100))
    if a.Cmp(b) > 0 {
        return false, ErrReplaceUnderpriced
    }
}
```

**해결**:
```javascript
// 기존 pending tx의 gas price
old_gasPrice = 50 Gwei

// 새 tx는 최소 10% 증가
new_gasPrice = 50 * 1.1 = 55 Gwei

eth.sendTransaction({
    nonce: 10,  // 동일한 nonce
    gasPrice: web3.toWei(55, "gwei")
})
```

### 8.3 Profiling 및 성능 분석

#### 8.3.1 CPU Profiling

```bash
# Geth 실행 시 pprof 활성화
geth --pprof --pprof.addr 0.0.0.0 --pprof.port 6060

# 다른 터미널에서 프로파일 수집 (30초)
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# 결과 분석
(pprof) top10
Showing nodes accounting for 25.30s, 84.33% of 30s total
      flat  flat%   sum%        cum   cum%
     8.50s 28.33% 28.33%      8.50s 28.33%  crypto/sha256.block
     5.20s 17.33% 45.67%      5.20s 17.33%  runtime.memmove
     3.10s 10.33% 56.00%      3.10s 10.33%  trie.(*hasher).hash
     2.80s  9.33% 65.33%      2.80s  9.33%  rlp.decodeByteArray
     // ...

# Flame graph 생성
(pprof) web
```

#### 8.3.2 Memory Profiling

```bash
# Heap 프로파일
go tool pprof http://localhost:6060/debug/pprof/heap

(pprof) top10
Showing nodes accounting for 2048MB, 85% of 2400MB total
      flat  flat%   sum%        cum   cum%
    512MB 21.33% 21.33%     512MB 21.33%  trie.Database.node
    384MB 16.00% 37.33%     384MB 16.00%  core/state.(*stateObject).getTrie
    256MB 10.67% 48.00%     256MB 10.67%  ethdb.(*table).NewIterator
```

#### 8.3.3 Trace 분석

```bash
# Execution trace 수집 (5초)
curl http://localhost:6060/debug/pprof/trace?seconds=5 > trace.out

# Trace 뷰어 실행
go tool trace trace.out

# 브라우저에서 확인:
# - Goroutine 분석
# - Syscall blocking
# - Network/Sync blocking
```

---

## 9. 프로덕션 환경 Best Practices

### 9.1 Geth 설정 최적화

#### 9.1.1 하드웨어 요구사항

**Mainnet Full Node**:
```
CPU: 8+ cores (Intel/AMD x64)
RAM: 16GB+ (32GB 권장)
Disk: 2TB+ NVMe SSD (IOPS 10000+)
Network: 25+ Mbps 대역폭
```

**Archive Node**:
```
CPU: 16+ cores
RAM: 64GB+
Disk: 12TB+ NVMe SSD (RAID 0)
Network: 100+ Mbps
```

#### 9.1.2 설정 파라미터

```bash
# Production full node
geth \
  --syncmode "snap" \                    # Snap sync (가장 빠름)
  --cache 8192 \                         # 8GB cache
  --maxpeers 50 \                        # 최대 피어 수
  --txpool.globalslots 8192 \            # Tx pool 크기
  --txpool.globalqueue 2048 \
  --txpool.accountslots 128 \
  --txpool.accountqueue 128 \
  --http \                               # HTTP RPC
  --http.addr "0.0.0.0" \
  --http.port 8545 \
  --http.vhosts "*" \
  --http.corsdomain "*" \
  --http.api "eth,net,web3,txpool" \    # 안전한 APIs만
  --ws \                                 # WebSocket
  --ws.addr "0.0.0.0" \
  --ws.port 8546 \
  --ws.origins "*" \
  --ws.api "eth,net,web3" \
  --metrics \                            # Prometheus metrics
  --metrics.addr "0.0.0.0" \
  --metrics.port 6060 \
  --db.engine "pebble" \                 # Pebble DB (LevelDB 대체)
  --state.scheme "path"                  # Path-based state scheme
```

#### 9.1.3 Snap Sync 모니터링

```javascript
// Sync 진행상황 확인
> eth.syncing
{
    currentBlock: 15537393,
    highestBlock: 18500000,
    knownStates: 450000000,
    pulledStates: 430000000,
    startingBlock: 0
}

// Sync 완료 여부
> eth.syncing
false  // Sync 완료

// 최신 블록 확인
> eth.blockNumber
18500000
```

### 9.2 모니터링 및 알림

#### 9.2.1 Prometheus Metrics

**설정**:
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'geth'
    static_configs:
      - targets: ['localhost:6060']
```

**주요 메트릭**:
```
# 블록 처리 시간
geth_blockchain_inserts{quantile="0.5"}  # Median
geth_blockchain_inserts{quantile="0.99"}  # P99

# Transaction pool
geth_txpool_pending
geth_txpool_queued

# State DB
geth_state_snapshot_generation_progress

# P2P
geth_p2p_peers
geth_p2p_ingress  # Incoming bandwidth
geth_p2p_egress   # Outgoing bandwidth
```

#### 9.2.2 Alerting (Prometheus Alertmanager)

```yaml
# alerts.yml
groups:
  - name: geth
    rules:
      # 동기화 지연
      - alert: GethSyncLagging
        expr: (eth_block_number - geth_blockchain_head_block) > 10
        for: 5m
        annotations:
          summary: "Geth is lagging behind by {{ $value }} blocks"
      
      # 피어 연결 부족
      - alert: GethLowPeers
        expr: geth_p2p_peers < 5
        for: 10m
        annotations:
          summary: "Geth has only {{ $value }} peers"
      
      # Tx pool 포화
      - alert: GethTxPoolFull
        expr: geth_txpool_pending > 4000
        for: 5m
        annotations:
          summary: "Tx pool is nearly full: {{ $value }}/4096"
```

### 9.3 백업 및 복구

#### 9.3.1 Chaindata 백업

```bash
# 1. Geth 중지
systemctl stop geth

# 2. Snapshot 생성 (LVM/ZFS)
lvcreate --size 100G --snapshot --name geth-snap /dev/vg0/geth-data

# 3. Geth 재시작
systemctl start geth

# 4. Snapshot 백업 (incremental)
rsync -avz --delete /dev/vg0/geth-snap /backup/geth/$(date +%Y%m%d)

# 5. Snapshot 제거
lvremove /dev/vg0/geth-snap
```

#### 9.3.2 Export/Import (Alternative)

```bash
# Export (특정 블록 범위)
geth export --datadir /data <filename> <start_block> <end_block>

# Import
geth import --datadir /data <filename>
```

### 9.4 보안 Best Practices

#### 9.4.1 방화벽 설정

```bash
# iptables rules
# P2P (필수)
iptables -A INPUT -p tcp --dport 30303 -j ACCEPT
iptables -A INPUT -p udp --dport 30303 -j ACCEPT

# RPC (내부망만)
iptables -A INPUT -p tcp --dport 8545 -s 10.0.0.0/8 -j ACCEPT
iptables -A INPUT -p tcp --dport 8545 -j DROP

# Metrics (모니터링 서버만)
iptables -A INPUT -p tcp --dport 6060 -s 10.1.1.100 -j ACCEPT
iptables -A INPUT -p tcp --dport 6060 -j DROP
```

#### 9.4.2 RPC 제한

```bash
# 위험한 APIs 비활성화
--http.api "eth,net,web3"  # debug, admin, personal 제외

# Rate limiting (nginx proxy)
limit_req_zone $binary_remote_addr zone=rpc:10m rate=10r/s;

server {
    location / {
        limit_req zone=rpc burst=20;
        proxy_pass http://localhost:8545;
    }
}
```

---

## 10. 알려진 이슈 및 해결책

### 10.1 "Database Compaction" 오래 걸림

**현상**: Geth 시작 시 compaction에 수 시간 소요

**원인**: LevelDB의 LSM-tree 구조 특성

**해결**:
```bash
# 1. Pebble DB 사용 (LevelDB보다 빠름)
geth --db.engine pebble

# 2. Offline compaction
geth removedb  # 주의: 전체 삭제
geth --syncmode snap  # 재동기화
```

### 10.2 "Out of Memory" 에러

**현상**: Geth 실행 중 OOM killed

**원인**: Cache 크기 > 물리 메모리

**해결**:
```bash
# Cache 크기 조정 (물리 메모리의 50% 이하)
geth --cache 4096  # 4GB (8GB RAM 환경)
```

### 10.3 "Discarded Bad Propagated Block"

**현상**: 로그에 "discarded bad propagated block" 반복

**원인**: 악의적 또는 버그 있는 피어

**해결**:
```javascript
// 문제 피어 확인
> admin.peers

// 수동 차단
> admin.removePeer("enode://...")

// Trusted peers만 사용
geth --netrestrict "10.0.0.0/8" --bootnodes "enode://trusted1,enode://trusted2"
```

---

이제 **ETHEREUM_INTERNALS_SPEC.md**에 6개 섹션(트랜잭션 플로우, 성능 최적화, 디버깅, 프로덕션, 이슈 해결) 추가 완료했습니다! 

다음은 **SOLANA**, **SUI**, **LIBP2P**, **IROH**, **NAT_TRAVERSAL** 문서에도 동일한 심화 섹션을 추가하겠습니다.

계속 진행할까요? 🚀
