// simple-merkle-tree.go
// Ethereum Merkle Patricia Trie의 기본 개념을 이해하기 위한 간단한 Merkle Tree 구현

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// MerkleNode: Merkle tree의 노드
type MerkleNode struct {
	Left  *MerkleNode // 왼쪽 자식
	Right *MerkleNode // 오른쪽 자식
	Data  []byte      // 노드 데이터 (해시값)
}

// MerkleTree: Merkle tree 구조
type MerkleTree struct {
	Root *MerkleNode // 루트 노드
}

// NewMerkleNode: 새로운 Merkle 노드 생성
// 목적: 리프 노드 또는 브랜치 노드 생성
func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	node := &MerkleNode{}

	// 리프 노드인 경우
	if left == nil && right == nil {
		// 데이터를 직접 해시
		hash := sha256.Sum256(data)
		node.Data = hash[:]
	} else {
		// 브랜치 노드인 경우
		// 좌우 자식의 해시를 합쳐서 해시
		prevHashes := append(left.Data, right.Data...)
		hash := sha256.Sum256(prevHashes)
		node.Data = hash[:]
	}

	node.Left = left
	node.Right = right

	return node
}

// NewMerkleTree: Merkle tree 생성
// 목적: 트랜잭션 리스트로부터 Merkle tree 구축
func NewMerkleTree(data [][]byte) *MerkleTree {
	var nodes []*MerkleNode

	// 1단계: 모든 리프 노드 생성
	// 의도: 각 트랜잭션을 리프 노드로 변환
	for _, datum := range data {
		node := NewMerkleNode(nil, nil, datum)
		nodes = append(nodes, node)
	}

	// 홀수 개의 노드가 있으면 마지막 노드 복제
	// 목적: 트리를 완전 이진 트리로 만들기
	if len(nodes)%2 != 0 {
		nodes = append(nodes, nodes[len(nodes)-1])
	}

	// 2단계: 상위 레벨 구축
	// 의도: 리프 노드부터 루트까지 bottom-up으로 구축
	for len(nodes) > 1 {
		var level []*MerkleNode

		// 두 개씩 묶어서 부모 노드 생성
		for i := 0; i < len(nodes); i += 2 {
			node := NewMerkleNode(nodes[i], nodes[i+1], nil)
			level = append(level, node)
		}

		// 다음 레벨 준비
		nodes = level

		// 홀수 개 처리
		if len(nodes)%2 != 0 && len(nodes) > 1 {
			nodes = append(nodes, nodes[len(nodes)-1])
		}
	}

	// 3단계: 트리 생성
	tree := &MerkleTree{nodes[0]}
	return tree
}

// VerifyProof: Merkle proof 검증
// 목적: 특정 데이터가 트리에 포함되어 있는지 검증
func VerifyProof(root []byte, data []byte, proof [][]byte, index int) bool {
	// 1. 데이터 해시 계산
	hash := sha256.Sum256(data)
	currentHash := hash[:]

	// 2. Proof 경로를 따라 루트까지 해시 계산
	for i, proofElement := range proof {
		var combined []byte

		// index의 비트를 확인하여 왼쪽/오른쪽 결정
		// 짝수 인덱스: 왼쪽, 홀수: 오른쪽
		if (index>>i)&1 == 0 {
			// 왼쪽: current + proof
			combined = append(currentHash, proofElement...)
		} else {
			// 오른쪽: proof + current
			combined = append(proofElement, currentHash...)
		}

		// 다음 레벨 해시 계산
		nextHash := sha256.Sum256(combined)
		currentHash = nextHash[:]
	}

	// 3. 계산된 루트와 실제 루트 비교
	for i := range root {
		if root[i] != currentHash[i] {
			return false
		}
	}

	return true
}

// printTree: 트리 구조 출력 (디버깅용)
func printTree(node *MerkleNode, prefix string, isTail bool) {
	if node == nil {
		return
	}

	// 현재 노드 출력
	fmt.Print(prefix)
	if isTail {
		fmt.Print("└── ")
	} else {
		fmt.Print("├── ")
	}
	fmt.Println(hex.EncodeToString(node.Data)[:8] + "...")

	// 자식 노드 출력
	if node.Left != nil || node.Right != nil {
		newPrefix := prefix
		if isTail {
			newPrefix += "    "
		} else {
			newPrefix += "│   "
		}

		if node.Right != nil {
			printTree(node.Right, newPrefix, false)
		}
		if node.Left != nil {
			printTree(node.Left, newPrefix, true)
		}
	}
}

func main() {
	fmt.Println("=== Merkle Tree 예제 ===\n")

	// 1. 트랜잭션 데이터 (실제로는 트랜잭션 해시)
	data := [][]byte{
		[]byte("tx1: Alice -> Bob: 10 ETH"),
		[]byte("tx2: Bob -> Charlie: 5 ETH"),
		[]byte("tx3: Charlie -> Dave: 3 ETH"),
		[]byte("tx4: Dave -> Alice: 2 ETH"),
	}

	fmt.Println("트랜잭션 데이터:")
	for i, tx := range data {
		fmt.Printf("%d: %s\n", i, string(tx))
	}

	// 2. Merkle tree 생성
	tree := NewMerkleTree(data)

	fmt.Printf("\nMerkle Root: %s\n\n", hex.EncodeToString(tree.Root.Data))

	// 3. 트리 구조 출력
	fmt.Println("트리 구조:")
	printTree(tree.Root, "", true)

	// 4. Merkle Proof 생성 (수동)
	// 실제로는 자동으로 생성해야 하지만, 예제를 위해 수동으로 구성
	// tx1의 proof: [hash(tx2), hash(tx3+tx4)]
	tx2Hash := sha256.Sum256(data[1])

	// tx3와 tx4의 부모 해시
	tx3Hash := sha256.Sum256(data[2])
	tx4Hash := sha256.Sum256(data[3])
	tx34Combined := append(tx3Hash[:], tx4Hash[:]...)
	tx34Hash := sha256.Sum256(tx34Combined)

	proof := [][]byte{
		tx2Hash[:],
		tx34Hash[:],
	}

	fmt.Println("\n=== Merkle Proof 검증 ===")
	fmt.Println("검증할 트랜잭션: tx1")
	fmt.Println("Proof:")
	for i, p := range proof {
		fmt.Printf("  Level %d: %s\n", i, hex.EncodeToString(p)[:16]+"...")
	}

	// 5. Proof 검증
	isValid := VerifyProof(tree.Root.Data, data[0], proof, 0)
	fmt.Printf("\n검증 결과: %v\n", isValid)

	if isValid {
		fmt.Println("✓ tx1이 이 블록에 포함되어 있음이 증명되었습니다!")
	} else {
		fmt.Println("✗ 검증 실패")
	}

	// 6. 잘못된 데이터로 검증 시도
	fmt.Println("\n=== 잘못된 데이터로 검증 ===")
	fakeData := []byte("tx1: Alice -> Bob: 100 ETH") // 금액 변조
	isValid = VerifyProof(tree.Root.Data, fakeData, proof, 0)
	fmt.Printf("검증 결과: %v\n", isValid)

	if !isValid {
		fmt.Println("✓ 변조된 데이터가 정상적으로 거부되었습니다!")
	}
}

/*
실행 방법:
go run simple-merkle-tree.go

예상 출력:
=== Merkle Tree 예제 ===

트랜잭션 데이터:
0: tx1: Alice -> Bob: 10 ETH
1: tx2: Bob -> Charlie: 5 ETH
2: tx3: Charlie -> Dave: 3 ETH
3: tx4: Dave -> Alice: 2 ETH

Merkle Root: 7a3c5b2f...

트리 구조:
└── 7a3c5b2f...
    ├── d4e5f6a7...
    │   ├── a1b2c3d4...
    │   └── e5f6a7b8...
    └── 8b9c0d1e...
        ├── 2f3a4b5c...
        └── 6d7e8f9a...

=== Merkle Proof 검증 ===
검증할 트랜잭션: tx1
Proof:
  Level 0: e5f6a7b8...
  Level 1: 8b9c0d1e...

검증 결과: true
✓ tx1이 이 블록에 포함되어 있음이 증명되었습니다!

=== 잘못된 데이터로 검증 ===
검증 결과: false
✓ 변조된 데이터가 정상적으로 거부되었습니다!

학습 포인트:
1. Merkle tree는 트랜잭션들을 계층적으로 해시하여 하나의 루트 해시를 생성
2. Merkle proof를 통해 특정 트랜잭션의 포함 여부를 O(log n)으로 검증 가능
3. 데이터 변조시 루트 해시가 달라지므로 무결성 보장
4. Ethereum의 Transaction Trie, Receipt Trie 등이 이 원리를 확장한 것

다음 단계:
- Patricia Trie 구현 (경로 압축)
- Ethereum의 실제 Merkle Patricia Trie 코드 분석 (trie/trie.go)
- RLP 인코딩 추가
*/
