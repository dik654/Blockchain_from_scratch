# 블록체인 실습 예제

각 블록체인의 핵심 개념을 이해하기 위한 실습 예제 모음입니다.

## 📁 구조

```
examples/
├── ethereum/          # Ethereum 예제
│   └── simple-merkle-tree.go
├── solana/            # Solana 예제
│   └── simple-poh.rs
└── sui/               # Sui 예제
    └── simple-coin.move
```

## 🎯 예제 설명

### Ethereum: Merkle Tree

**파일**: `ethereum/simple-merkle-tree.go`

**학습 목표**:
- Merkle tree의 기본 구조 이해
- Merkle proof 생성 및 검증
- Ethereum의 Transaction/Receipt Trie 기초

**핵심 개념**:
```
트랜잭션들 → Merkle Tree → Root Hash
                ↓
            Merkle Proof
                ↓
        O(log n) 검증
```

**실행**:
```bash
cd ethereum
go run simple-merkle-tree.go
```

**학습 포인트**:
1. 블록의 모든 트랜잭션을 하나의 해시로 압축
2. 특정 트랜잭션 포함 여부를 빠르게 증명
3. 데이터 무결성 보장 (변조시 루트 해시 변경)

**다음 단계**:
- Patricia Trie 구현 (경로 압축)
- RLP 인코딩 추가
- go-ethereum의 실제 코드 분석 (`trie/trie.go`)

---

### Solana: Proof of History

**파일**: `solana/simple-poh.rs`

**학습 목표**:
- PoH (시간 증명)의 원리 이해
- SHA-256 해시 체인
- 트랜잭션 순서 보장 메커니즘

**핵심 개념**:
```
Hash_0 = SHA256(seed)
Hash_1 = SHA256(Hash_0)
Hash_2 = SHA256(Hash_1)
...
Hash_n = SHA256(Hash_{n-1})

→ 순차적으로만 계산 가능
→ 시간 경과 증명
```

**실행**:
```bash
cd solana
cargo new simple-poh
cd simple-poh
# Cargo.toml에 sha2 = "0.10" 추가
# src/main.rs에 코드 복사
cargo run
```

**학습 포인트**:
1. 시간의 암호학적 증명 (Verifiable Delay Function)
2. 트랜잭션 순서가 PoH 스트림에 영구 기록
3. 순서 변조 불가능 (해시 체인)
4. Solana 고성능의 핵심 기술

**다음 단계**:
- Tower BFT 합의와의 통합
- Leader 스케줄링
- Solana 실제 코드 (`poh/src/poh_recorder.rs`)

---

### Sui: Simple Coin (객체 모델)

**파일**: `sui/simple-coin.move`

**학습 목표**:
- Sui의 객체 중심 모델 이해
- 소유권 타입 (Address, Shared, Object, Immutable)
- Move 언어 기초
- FastPath vs ConsensusPath

**핵심 개념**:
```
Object {
    id: UID,           // 고유 ID
    owner: Owner,      // 소유자
    data: ...          // 데이터
}

소유 객체 → FastPath (합의 불필요!)
공유 객체 → ConsensusPath (합의 필요)
```

**실행**:
```bash
cd sui

# Sui 설치
cargo install --locked --git https://github.com/MystenLabs/sui.git --branch mainnet sui

# 모듈 빌드 (프로젝트 생성 필요)
sui move build

# 배포
sui client publish --gas-budget 50000000
```

**학습 포인트**:
1. **객체 중심**: 모든 것이 객체 (Wallet, Coin 등)
2. **명확한 소유권**: 누가 무엇을 소유하는지 명확
3. **병렬 실행**: 소유 객체는 독립적으로 처리
4. **Move semantics**: 안전한 리소스 관리

**다음 단계**:
- 복잡한 객체 관계 (child objects, dynamic fields)
- Shared 객체 최적화
- Sui 실제 코드 (`crates/sui-framework/`)

---

## 🚀 학습 순서

### 1단계: 기초 개념 (1주)
```
1. Ethereum Merkle Tree 실행
2. 코드 읽으며 주석 이해
3. 직접 수정해보기 (다른 해시 함수 등)
```

### 2단계: 고급 개념 (1주)
```
1. Solana PoH 실행
2. 시간 증명 원리 이해
3. tick 개수 변경하며 실험
```

### 3단계: 최신 기술 (1주)
```
1. Sui Coin 배포
2. 객체 모델 이해
3. FastPath vs ConsensusPath 비교
```

## 💡 실습 팁

### 코드 수정 아이디어

**Merkle Tree**:
- [ ] 다른 해시 함수 사용 (SHA-512, Blake2b)
- [ ] 4-ary tree 구현 (자식 4개)
- [ ] Sparse Merkle Tree
- [ ] 증분 업데이트

**PoH**:
- [ ] hashes_per_tick 변경하며 성능 측정
- [ ] 멀티스레드 검증
- [ ] GPU 가속 (CUDA)
- [ ] VDF (Verifiable Delay Function) 비교

**Sui Coin**:
- [ ] NFT 구현
- [ ] Staking 메커니즘
- [ ] DAO (투표)
- [ ] DEX (간단한 AMM)

### 디버깅

**Go (Ethereum)**:
```bash
# 디버그 정보 출력
GODEBUG=gctrace=1 go run simple-merkle-tree.go

# 프로파일링
go run simple-merkle-tree.go -cpuprofile=cpu.prof
go tool pprof cpu.prof
```

**Rust (Solana)**:
```bash
# 디버그 빌드
cargo build

# 릴리스 빌드 (최적화)
cargo build --release

# 테스트
cargo test

# 벤치마크
cargo bench
```

**Move (Sui)**:
```bash
# 테스트
sui move test

# 커버리지
sui move test --coverage

# Prover (formal verification)
sui move prove
```

## 📚 추가 학습 자료

### Ethereum
- [Ethereum Yellow Paper](https://ethereum.github.io/yellowpaper/paper.pdf)
- [Merkle Patricia Trie 설명](https://ethereum.org/en/developers/docs/data-structures-and-encoding/patricia-merkle-trie/)
- [go-ethereum 코드](https://github.com/ethereum/go-ethereum)

### Solana
- [Proof of History 논문](https://solana.com/solana-whitepaper.pdf)
- [Solana Cookbook](https://solanacookbook.com)
- [Agave 코드](https://github.com/anza-xyz/agave)

### Sui
- [Sui Documentation](https://docs.sui.io)
- [Move Book](https://move-book.com)
- [Sui 코드](https://github.com/MystenLabs/sui)

## 🤝 기여

예제 개선이나 새로운 예제 추가를 환영합니다!

**추가하면 좋을 예제**:
- [ ] Ethereum: Simple ERC-20
- [ ] Ethereum: Gas 최적화 패턴
- [ ] Solana: Account model 예제
- [ ] Solana: Anchor 프로그램
- [ ] Sui: Dynamic fields
- [ ] Sui: Capability 패턴

## ⚠️ 주의사항

- 이 예제들은 **교육 목적**입니다
- 프로덕션 사용 금지
- 보안 감사 없음
- 실제 코드는 훨씬 복잡합니다

## 📞 질문

예제 관련 질문은 GitHub Issues에 올려주세요.

---

**Happy Coding! 🚀**
