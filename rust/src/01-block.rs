// 타임스탬프를 찍기위한 라이브러리
use std::time::SystemTime;
// 해싱용 라이브러리
use crypto::digest::Digest;
use crypto::sha2::Sha256;
// 로깅용
use log::info;

// 제네릭 선언
pub type Result<T> = std::result::Result<T, failure::Error>;
// PoW 난이도
const TARGET_HEXT: usize = 4;

// 객체를 복제하는 메소드 clone()을 정의
#[derive(Debug, Clone)]
// 블록체인은 이전 블록 구조체 정보를 해시값으로 담아 
// 리스트 자료구조 형식으로 데이터를 연결한다
pub struct Block {
    // 여러 트랜잭션들이 모인 블록이 생성된 시간
    timestamp: u128,
    // 트랜잭션 정보들
    transactions: String,
    // 이전 블록의 해시값
    prev_block_hash: String,
    // "다음 블록이 갖는 이전블록의 해시값"과 비교하기 위한 "현재 블록의 해시값"(검증용)
    hash: String,
    // 현재 블록까지 생성된 블록의 총 개수
    height: usize,
    // PoW에서 문제에 대한 해답
    nonce: i32,
}
#[derive(Debug)]
pub struct Blockchain {
    // 블록체인은 해시값으로 연결된 블록들의 연속
    blocks: Vec<Block>
}

// 블록 구조체의 메서드들을 선언
impl Block {
    // 첫 시작 블록 생성함수
    pub fn new_genesis_block() -> Block {
        Block::new_block(String::from("Genesis Block"), String::new(), 0).unwrap()
    }
    // 해시값 가져오기 clone으로 가져오는 이유는 소유권때문
    pub fn get_hash(&self) -> String {
        self.hash.clone()
    }
    // data는 블록으로 감쌀 트랜잭션들
    // pre_block_hash는 가장 최신의 블록의 hash값
    // height는 가장 최신의 블록의 height + 1
    pub fn new_block(data: String, prev_block_hash: String, height: usize) -> Result<Block> {
        // 타임스탬프로 생성하는 현재시간 찍기
        let timestamp: u128 = SystemTime::now()
            // unix형식으로 리턴(없을경우 2023-07-21 13:45:59.123456789 UTC같이 찍힌다) 
            .duration_since(SystemTime::UNIX_EPOCH)?
            .as_millis();
        // 블록 생성
        let mut block = Block {
            timestamp,
            transactions: data,
            prev_block_hash,
            hash: String::new(),
            height,
            // PoW 실행 전에는 0으로 초기화
            nonce: 0,
        };
        // PoW 실행
        block.run_proof_if_work()?;
        Ok(block)
    }
    // 인수는 파이썬의 self와 같은 역할
    // 메서드를 실행시키는 block 자신을 가리킴
    fn run_proof_if_work(&mut self) -> Result<()> {
        info!("Mining the block");
        // 숫자를 1씩 증가시켜가며 문제가 해결되는지 체크
        while !self.validate()? {
            self.nonce += 1;
        }
        // block의 정보가 바뀌었으므로 전체 해시값도 바뀌니 해싱할 준비
        // 블록 정보를 직렬화해서 가져오기
        let data = self.prepare_hash_data()?;
        // sha256로 해싱할 준비
        let mut hasher = Sha256::new();
        // hasher에 직렬화된 block 정보를 담아
        hasher.input(&data[..]);
        // 해싱 후 block 구조체의 hash 갱신
        self.hash = hasher.result_str();
        Ok(())
    }

    fn prepare_hash_data(&self) -> Result<Vec<u8>> {
        let content = (
            self.prev_block_hash.clone(),
            self.transactions.clone(),
            self.timestamp,
            TARGET_HEXT,
            self.nonce
        );
        // 직렬화해서 전달
        let bytes = bincode::serialize(&content)?;
        Ok(bytes)
    }

    // 리턴값으로 Result를 사용하면 에러처리를 편하게 할 수 있음
    // Ok()를 이용하여 리턴하는 것은 Result를 사용하기위함
    // PoW는 해싱을 했을 때 처음 나타나는 0의 개수로 난이도를 조절하는데
    // Ok()내부의 로직으로 이를 구현
    fn validate(&self) -> Result<bool> {
        let data = self.prepare_hash_data()?;
        let mut hasher = Sha256::new();
        hasher.input(&data[..]);
        let mut vec1: Vec<u8> = vec![];
        vec1.resize(TARGET_HEXT, '0' as u8);
        println!("{:?}", vec1);
        // 해싱한 결과에서 0000으로 시작하는지 검사
        Ok(&hasher.result_str()[0..TARGET_HEXT] == String::from_utf8(vec1)?)
    }
}

impl Blockchain {
    // 제네시스 블록으로 첫 블록체인 구조체 생성
    pub fn new() -> Blockchain {
        Blockchain {
            blocks: vec![Block::new_genesis_block()]
        }
    }
    
    pub fn add_block(&mut self, data: String) -> Result<()> {
        let prev = self.blocks.last().unwrap();
        let new_block = Block::new_block(data, prev.get_hash(), TARGET_HEXT)?;
        self.blocks.push(new_block);
        Ok(())
    }
}
// 테스트 코드
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_blockchain() {
        let mut b = Blockchain::new();
        b.add_block("data".to_string());
        b.add_block("data2".to_string()); 
        b.add_block("data3".to_string());
        dbg!(b);
    } 
}