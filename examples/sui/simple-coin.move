// simple-coin.move
// Sui의 객체 중심 모델을 이해하기 위한 간단한 Coin 구현
//
// 핵심 개념:
// 1. Object: Sui의 기본 저장 단위
// 2. Owner: 소유권 타입 (Address, Object, Shared, Immutable)
// 3. Transfer: 객체 전송
// 4. Programmable Transactions: 여러 명령을 조합

module examples::simple_coin {
    use sui::object::{Self, UID};
    use sui::tx_context::{Self, TxContext};
    use sui::transfer;
    use sui::coin::{Self, Coin};
    use sui::balance::{Self, Balance};

    // === 오류 코드 ===
    const EInsufficientBalance: u64 = 0;
    const EInvalidAmount: u64 = 1;

    // === Coin 타입 정의 ===
    // Coin<T>: 제네릭 타입으로 다양한 코인 표현
    // 예: Coin<SUI>, Coin<USDC>

    /// SimpleCoin: 우리만의 코인 타입
    /// 목적: 객체 모델 이해를 위한 예제
    struct SimpleCoin has drop {}

    // === 지갑 객체 ===
    /// Wallet: 코인을 보관하는 객체
    ///
    /// 특징:
    /// - key: 고유 ID를 가짐 (글로벌하게 접근 가능)
    /// - store: 다른 객체에 저장 가능
    struct Wallet has key, store {
        id: UID,
        balance: Balance<SimpleCoin>,
    }

    // === 초기화 함수 ===
    /// init: 모듈 배포시 한 번만 실행
    /// 목적: 초기 설정 (보통 Treasury 생성 등)
    fun init(ctx: &mut TxContext) {
        // 이 예제에서는 특별한 초기화 없음
        // 실제로는 TreasuryCap 생성 등을 수행
    }

    // === 지갑 생성 ===
    /// create_wallet: 새 지갑 생성
    ///
    /// 흐름:
    /// 1. 새로운 UID 생성 (고유 ID)
    /// 2. 빈 Balance로 Wallet 객체 생성
    /// 3. 발신자에게 전송 (AddressOwner)
    public entry fun create_wallet(ctx: &mut TxContext) {
        // 새로운 객체 생성
        let wallet = Wallet {
            id: object::new(ctx),  // 고유 ID 생성
            balance: balance::zero(),  // 빈 잔액
        };

        // 발신자에게 전송
        // 의도: 이 Wallet은 이제 발신자가 소유
        // 결과: Owner = AddressOwner(sender)
        transfer::transfer(wallet, tx_context::sender(ctx));
    }

    // === 코인 발행 (Mint) ===
    /// mint: 새로운 코인 발행
    ///
    /// 주의: 실제로는 TreasuryCap 권한 필요
    /// 여기서는 단순화를 위해 누구나 발행 가능하게 구현
    ///
    /// 흐름:
    /// 1. 새 Balance 생성
    /// 2. 지정된 금액 추가
    /// 3. Coin 객체로 변환
    /// 4. 수신자에게 전송
    public entry fun mint(
        amount: u64,
        recipient: address,
        ctx: &mut TxContext
    ) {
        // 금액 검증
        assert!(amount > 0, EInvalidAmount);

        // 새 Balance 생성
        let balance = balance::create_for_testing<SimpleCoin>(amount);

        // Coin 객체 생성
        let coin = coin::from_balance(balance, ctx);

        // 수신자에게 전송
        transfer::public_transfer(coin, recipient);
    }

    // === 코인 전송 ===
    /// transfer_coin: 코인을 다른 주소로 전송
    ///
    /// 특징:
    /// - coin 객체를 소비 (move semantics)
    /// - 수신자는 새로운 소유자가 됨
    /// - 트랜잭션 실패시 자동 롤백
    public entry fun transfer_coin(
        coin: Coin<SimpleCoin>,
        recipient: address,
        _ctx: &mut TxContext
    ) {
        // Coin 객체를 수신자에게 전송
        // 의도: 소유권 이전
        transfer::public_transfer(coin, recipient);
    }

    // === 코인 분할 ===
    /// split_coin: 코인을 두 개로 분할
    ///
    /// 흐름:
    /// 1. 원본 코인에서 지정 금액 분리
    /// 2. 새 코인 객체 생성
    /// 3. 수신자에게 전송
    /// 4. 원본 코인은 발신자 보유 (잔액 감소)
    public entry fun split_coin(
        coin: &mut Coin<SimpleCoin>,
        amount: u64,
        recipient: address,
        ctx: &mut TxContext
    ) {
        // 잔액 확인
        assert!(coin::value(coin) >= amount, EInsufficientBalance);

        // 코인 분할
        let split = coin::split(coin, amount, ctx);

        // 분할된 코인을 수신자에게
        transfer::public_transfer(split, recipient);

        // 원본 코인은 자동으로 발신자에게 반환 (mut 참조)
    }

    // === 코인 병합 ===
    /// merge_coins: 두 코인을 하나로 병합
    ///
    /// 흐름:
    /// 1. coin2의 잔액을 coin1에 합침
    /// 2. coin2는 소멸
    /// 3. coin1은 증가된 잔액으로 발신자에게
    public entry fun merge_coins(
        coin1: &mut Coin<SimpleCoin>,
        coin2: Coin<SimpleCoin>,
        _ctx: &mut TxContext
    ) {
        // 병합
        // 의도: coin2의 Balance를 coin1에 추가하고 coin2 삭제
        coin::join(coin1, coin2);

        // coin1은 자동으로 발신자에게 반환 (mut 참조)
    }

    // === 지갑에 입금 ===
    /// deposit: 지갑에 코인 입금
    ///
    /// 흐름:
    /// 1. Coin을 Balance로 변환
    /// 2. Wallet의 balance에 합침
    /// 3. Coin 객체는 소멸
    public entry fun deposit(
        wallet: &mut Wallet,
        coin: Coin<SimpleCoin>,
        _ctx: &mut TxContext
    ) {
        // Coin을 Balance로 변환
        let balance_to_add = coin::into_balance(coin);

        // Wallet에 추가
        balance::join(&mut wallet.balance, balance_to_add);
    }

    // === 지갑에서 출금 ===
    /// withdraw: 지갑에서 코인 출금
    ///
    /// 흐름:
    /// 1. Wallet에서 Balance 분리
    /// 2. Balance를 Coin으로 변환
    /// 3. 수신자에게 전송
    public entry fun withdraw(
        wallet: &mut Wallet,
        amount: u64,
        recipient: address,
        ctx: &mut TxContext
    ) {
        // 잔액 확인
        assert!(balance::value(&wallet.balance) >= amount, EInsufficientBalance);

        // Balance 분리
        let withdrawn_balance = balance::split(&mut wallet.balance, amount);

        // Coin으로 변환
        let coin = coin::from_balance(withdrawn_balance, ctx);

        // 수신자에게 전송
        transfer::public_transfer(coin, recipient);
    }

    // === 조회 함수 ===
    /// wallet_balance: 지갑 잔액 조회
    public fun wallet_balance(wallet: &Wallet): u64 {
        balance::value(&wallet.balance)
    }

    // === 공유 객체 예제 ===
    /// SharedWallet: 공유 지갑 (누구나 접근 가능)
    ///
    /// 특징:
    /// - Owner = Shared
    /// - 합의 필요 (ConsensusPath)
    /// - 여러 트랜잭션이 동시에 접근 불가 (순차 실행)
    struct SharedWallet has key {
        id: UID,
        balance: Balance<SimpleCoin>,
    }

    /// create_shared_wallet: 공유 지갑 생성
    ///
    /// 의도: 누구나 입출금 가능한 지갑
    /// 사용 예: DAO 금고, 공용 풀
    public entry fun create_shared_wallet(ctx: &mut TxContext) {
        let shared_wallet = SharedWallet {
            id: object::new(ctx),
            balance: balance::zero(),
        };

        // 공유 객체로 전환
        // 결과: Owner = Shared
        // 주의: 이후 합의 필요!
        transfer::share_object(shared_wallet);
    }

    /// deposit_to_shared: 공유 지갑에 입금
    public entry fun deposit_to_shared(
        shared_wallet: &mut SharedWallet,
        coin: Coin<SimpleCoin>,
        _ctx: &mut TxContext
    ) {
        let balance_to_add = coin::into_balance(coin);
        balance::join(&mut shared_wallet.balance, balance_to_add);
    }

    /// withdraw_from_shared: 공유 지갑에서 출금
    public entry fun withdraw_from_shared(
        shared_wallet: &mut SharedWallet,
        amount: u64,
        recipient: address,
        ctx: &mut TxContext
    ) {
        assert!(balance::value(&shared_wallet.balance) >= amount, EInsufficientBalance);

        let withdrawn_balance = balance::split(&mut shared_wallet.balance, amount);
        let coin = coin::from_balance(withdrawn_balance, ctx);

        transfer::public_transfer(coin, recipient);
    }
}

/*
사용 예제 (Sui CLI):

1. 모듈 배포:
sui client publish --gas-budget 50000000

2. 지갑 생성:
sui client call \
  --package <PACKAGE_ID> \
  --module simple_coin \
  --function create_wallet \
  --gas-budget 10000000

3. 코인 발행:
sui client call \
  --package <PACKAGE_ID> \
  --module simple_coin \
  --function mint \
  --args 1000 <YOUR_ADDRESS> \
  --gas-budget 10000000

4. 코인 전송:
sui client call \
  --package <PACKAGE_ID> \
  --module simple_coin \
  --function transfer_coin \
  --args <COIN_OBJECT_ID> <RECIPIENT_ADDRESS> \
  --gas-budget 10000000

5. Programmable Transaction (여러 명령 조합):
sui client ptb \
  --split-coins gas [1000] \
  --assign new_coin \
  --transfer-objects [new_coin] @<recipient> \
  --gas-budget 10000000

학습 포인트:

1. 객체 모델:
   - Wallet, Coin 모두 객체 (고유 ID)
   - 각 객체는 명확한 소유자
   - 객체는 생성, 전송, 삭제 가능

2. 소유권 타입:
   - AddressOwner: 특정 주소 소유 (FastPath - 합의 불필요!)
   - Shared: 공유 (ConsensusPath - 합의 필요)
   - ObjectOwner: 다른 객체 소유
   - Immutable: 불변

3. Move Semantics:
   - 객체는 소비되거나 참조로 전달
   - 소비된 객체는 자동 삭제 또는 전송
   - mut 참조는 자동으로 발신자에게 반환

4. 병렬 실행:
   - AddressOwner 객체만 사용 -> 병렬 실행 가능!
   - Shared 객체 포함 -> 합의 필요, 순차 실행

5. Programmable Transactions:
   - 여러 명령을 원자적으로 실행
   - 중간 결과 전달 가능
   - 복잡한 로직을 온체인에서 조합

다음 단계:
- Sui의 실제 Coin 표준 분석 (sui-framework/sources/coin.move)
- 복잡한 객체 관계 (child objects, dynamic fields)
- Shared 객체 최적화 기법
- Move 프로버 (formal verification)
*/
