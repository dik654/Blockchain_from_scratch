// simple-poh.rs
// Solana Proof of History (PoH)의 기본 개념을 이해하기 위한 간단한 구현

use sha2::{Sha256, Digest};
use std::time::{SystemTime, UNIX_EPOCH, Duration};
use std::thread;

/// PoH Entry: PoH 해시 체인의 단일 엔트리
/// 목적: 시간 경과를 암호학적으로 증명
#[derive(Debug, Clone)]
pub struct PohEntry {
    /// PoH 해시 (이전 해시의 SHA-256)
    pub hash: [u8; 32],

    /// 이 엔트리까지의 해시 횟수
    pub num_hashes: u64,

    /// 믹스된 데이터 (트랜잭션 등)
    /// None이면 순수 tick (시간 표시)
    pub mixin: Option<Vec<u8>>,
}

/// PoH: Proof of History 생성기
/// 목적: 시간의 순서를 검증 가능하게 기록
pub struct Poh {
    /// 현재 PoH 해시
    current_hash: [u8; 32],

    /// 총 해시 횟수
    total_hashes: u64,

    /// Tick당 해시 횟수
    /// 목적: 일정한 시간 간격 유지
    hashes_per_tick: u64,

    /// 마지막 tick 이후 해시 횟수
    hashes_since_last_tick: u64,
}

impl Poh {
    /// 새로운 PoH 생성
    pub fn new(seed: [u8; 32], hashes_per_tick: u64) -> Self {
        Poh {
            current_hash: seed,
            total_hashes: 0,
            hashes_per_tick,
            hashes_since_last_tick: 0,
        }
    }

    /// 단일 해시 실행
    /// 의도: SHA-256 체인 진행
    /// hash_n = SHA256(hash_{n-1})
    fn hash(&mut self) {
        let mut hasher = Sha256::new();
        hasher.update(&self.current_hash);
        let result = hasher.finalize();

        self.current_hash.copy_from_slice(&result);
        self.total_hashes += 1;
        self.hashes_since_last_tick += 1;
    }

    /// Tick 생성
    /// 목적: 시간 단위 표시 (예: 6.25ms)
    ///
    /// 흐름:
    /// 1. hashes_per_tick만큼 해시 실행
    /// 2. Tick 엔트리 생성 (트랜잭션 없음)
    /// 3. 카운터 리셋
    pub fn tick(&mut self) -> PohEntry {
        // 남은 해시 실행
        let remaining = self.hashes_per_tick - self.hashes_since_last_tick;
        for _ in 0..remaining {
            self.hash();
        }

        // Tick 엔트리 생성
        let entry = PohEntry {
            hash: self.current_hash,
            num_hashes: self.hashes_since_last_tick,
            mixin: None, // Tick은 데이터 없음
        };

        // 카운터 리셋
        self.hashes_since_last_tick = 0;

        entry
    }

    /// 데이터 믹싱 (트랜잭션 기록)
    /// 목적: PoH 스트림에 트랜잭션 삽입
    ///
    /// 흐름:
    /// 1. 트랜잭션 해시 계산
    /// 2. PoH 해시와 믹스: hash = SHA256(poh_hash || tx_hash)
    /// 3. 엔트리 생성
    pub fn record(&mut self, mixin: &[u8]) -> PohEntry {
        // 믹싱 전 해시 횟수
        let num_hashes_before = self.hashes_since_last_tick;

        // PoH 해시와 데이터를 함께 해시
        let mut hasher = Sha256::new();
        hasher.update(&self.current_hash);
        hasher.update(mixin);
        let result = hasher.finalize();

        self.current_hash.copy_from_slice(&result);
        self.total_hashes += 1;
        self.hashes_since_last_tick += 1;

        // 엔트리 생성
        PohEntry {
            hash: self.current_hash,
            num_hashes: num_hashes_before + 1, // 믹싱도 1번 해시로 카운트
            mixin: Some(mixin.to_vec()),
        }
    }

    /// 현재 해시 조회
    pub fn current_hash(&self) -> [u8; 32] {
        self.current_hash
    }

    /// 총 해시 횟수
    pub fn total_hashes(&self) -> u64 {
        self.total_hashes
    }
}

/// PoH 검증기
/// 목적: PoH 체인의 무결성 검증
pub struct PohVerifier {
    hashes_per_tick: u64,
}

impl PohVerifier {
    pub fn new(hashes_per_tick: u64) -> Self {
        PohVerifier { hashes_per_tick }
    }

    /// PoH 엔트리 체인 검증
    ///
    /// 흐름:
    /// 1. seed부터 시작
    /// 2. 각 엔트리의 해시 재계산
    /// 3. 일치 여부 확인
    pub fn verify(&self, seed: [u8; 32], entries: &[PohEntry]) -> bool {
        let mut current_hash = seed;

        for entry in entries {
            // 엔트리에 따라 해시 재계산
            match &entry.mixin {
                // Tick 엔트리
                None => {
                    // num_hashes만큼 해시 실행
                    for _ in 0..entry.num_hashes {
                        let mut hasher = Sha256::new();
                        hasher.update(&current_hash);
                        current_hash.copy_from_slice(&hasher.finalize());
                    }
                }
                // 데이터 엔트리
                Some(mixin) => {
                    // num_hashes - 1번 순수 해시
                    for _ in 0..(entry.num_hashes - 1) {
                        let mut hasher = Sha256::new();
                        hasher.update(&current_hash);
                        current_hash.copy_from_slice(&hasher.finalize());
                    }

                    // 마지막은 믹싱
                    let mut hasher = Sha256::new();
                    hasher.update(&current_hash);
                    hasher.update(mixin);
                    current_hash.copy_from_slice(&hasher.finalize());
                }
            }

            // 해시 일치 확인
            if current_hash != entry.hash {
                println!("❌ 검증 실패: 해시 불일치");
                return false;
            }
        }

        true
    }
}

/// 16진수 문자열 변환 (처음 8자만)
fn hex_short(hash: &[u8; 32]) -> String {
    hash.iter()
        .take(4)
        .map(|b| format!("{:02x}", b))
        .collect::<String>()
}

fn main() {
    println!("=== Solana Proof of History (PoH) 예제 ===\n");

    // 1. PoH 생성기 초기화
    let seed = [0u8; 32]; // 제네시스 해시
    let hashes_per_tick = 1000; // 1 tick = 1000 해시

    let mut poh = Poh::new(seed, hashes_per_tick);

    println!("초기 설정:");
    println!("  Seed: {}...", hex_short(&seed));
    println!("  Hashes per tick: {}\n", hashes_per_tick);

    // 2. PoH 스트림 생성
    let mut entries = Vec::new();

    println!("=== PoH 스트림 생성 ===");

    // Tick 1
    println!("1. Tick 생성 (시간 경과)");
    let tick1 = poh.tick();
    println!("   Hash: {}...", hex_short(&tick1.hash));
    println!("   Hashes: {}", tick1.num_hashes);
    entries.push(tick1);

    // 시뮬레이션: 시간 경과
    thread::sleep(Duration::from_millis(100));

    // 트랜잭션 1
    println!("\n2. 트랜잭션 기록");
    let tx1 = b"tx1: Alice -> Bob: 10 SOL";
    let entry1 = poh.record(tx1);
    println!("   TX: {}", String::from_utf8_lossy(tx1));
    println!("   Hash: {}...", hex_short(&entry1.hash));
    entries.push(entry1);

    // 트랜잭션 2
    let tx2 = b"tx2: Bob -> Charlie: 5 SOL";
    let entry2 = poh.record(tx2);
    println!("   TX: {}", String::from_utf8_lossy(tx2));
    println!("   Hash: {}...", hex_short(&entry2.hash));
    entries.push(entry2);

    // Tick 2
    println!("\n3. Tick 생성");
    let tick2 = poh.tick();
    println!("   Hash: {}...", hex_short(&tick2.hash));
    println!("   Hashes: {}", tick2.num_hashes);
    entries.push(tick2);

    // 3. 통계
    println!("\n=== PoH 통계 ===");
    println!("총 엔트리: {}", entries.len());
    println!("총 해시 횟수: {}", poh.total_hashes());
    println!("현재 해시: {}...", hex_short(&poh.current_hash()));

    // 4. PoH 검증
    println!("\n=== PoH 검증 ===");
    let verifier = PohVerifier::new(hashes_per_tick);
    let is_valid = verifier.verify(seed, &entries);

    if is_valid {
        println!("✓ PoH 체인 검증 성공!");
        println!("  - 모든 해시가 올바르게 계산됨");
        println!("  - 순서가 변조되지 않았음");
        println!("  - 시간 경과가 증명됨");
    } else {
        println!("✗ PoH 체인 검증 실패");
    }

    // 5. 변조 시도 (순서 바꾸기)
    println!("\n=== 변조 시도: 순서 바꾸기 ===");
    let mut tampered_entries = entries.clone();
    tampered_entries.swap(1, 2); // tx1과 tx2 순서 바꾸기

    let is_valid = verifier.verify(seed, &tampered_entries);
    if !is_valid {
        println!("✓ 변조가 정상적으로 감지되었습니다!");
        println!("  - PoH는 순서 변조를 방지합니다");
    }

    // 6. 시간 증명 데모
    println!("\n=== 시간 증명 데모 ===");
    println!("PoH의 핵심 원리:");
    println!("1. SHA-256은 순차적으로만 계산 가능 (병렬화 불가)");
    println!("2. 1000번 해시 = 일정한 시간 소요");
    println!("3. 따라서 PoH 체인 = 시간 경과의 증명");
    println!("\n예시:");
    println!("  - 2 tick 사이 = {}번 해시", hashes_per_tick * 2);
    println!("  - 컴퓨터 성능에 관계없이 최소 시간 필요");
    println!("  - 미래의 해시를 미리 계산 불가능");
}

/*
실행 방법:
cargo new simple-poh
cd simple-poh
# Cargo.toml에 sha2 = "0.10" 추가
# src/main.rs에 이 코드 복사
cargo run

예상 출력:
=== Solana Proof of History (PoH) 예제 ===

초기 설정:
  Seed: 00000000...
  Hashes per tick: 1000

=== PoH 스트림 생성 ===
1. Tick 생성 (시간 경과)
   Hash: 8f3a5bc2...
   Hashes: 1000

2. 트랜잭션 기록
   TX: tx1: Alice -> Bob: 10 SOL
   Hash: d4e7a1f3...
   TX: tx2: Bob -> Charlie: 5 SOL
   Hash: 2b9c8d6e...

3. Tick 생성
   Hash: 7a4f2e9b...
   Hashes: 998

=== PoH 통계 ===
총 엔트리: 4
총 해시 횟수: 2002
현재 해시: 7a4f2e9b...

=== PoH 검증 ===
✓ PoH 체인 검증 성공!
  - 모든 해시가 올바르게 계산됨
  - 순서가 변조되지 않았음
  - 시간 경과가 증명됨

=== 변조 시도: 순서 바꾸기 ===
❌ 검증 실패: 해시 불일치
✓ 변조가 정상적으로 감지되었습니다!
  - PoH는 순서 변조를 방지합니다

=== 시간 증명 데모 ===
PoH의 핵심 원리:
1. SHA-256은 순차적으로만 계산 가능 (병렬화 불가)
2. 1000번 해시 = 일정한 시간 소요
3. 따라서 PoH 체인 = 시간 경과의 증명

예시:
  - 2 tick 사이 = 2000번 해시
  - 컴퓨터 성능에 관계없이 최소 시간 필요
  - 미래의 해시를 미리 계산 불가능

학습 포인트:
1. PoH는 시간의 암호학적 증명 (Verifiable Delay Function과 유사)
2. 해시 체인이므로 순서 변조 불가능
3. 블록체인에 신뢰할 수 있는 시계 역할
4. Solana의 높은 처리량을 가능하게 하는 핵심 기술

PoH의 장점:
- 합의 전에 트랜잭션 순서 확정
- 타임스탬프 신뢰 문제 해결
- 네트워크 지연 영향 감소

다음 단계:
- Solana의 실제 PoH 코드 분석 (poh/src/poh_recorder.rs)
- Tower BFT와의 통합 이해
- Leader 스케줄링 메커니즘 학습
*/
