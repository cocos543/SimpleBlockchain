package main

import (
	"crypto/sha256"
)

type MerkleTree struct {
	RootNode *MerkleNode
}

type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
}

// NewMerkleNode 创建一棵树
func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	mNode := MerkleNode{}

	if left == nil && right == nil {
		hash := sha256.Sum256(data)
		mNode.Data = hash[:]
	} else {
		pervHash := append(left.Data, right.Data...)
		hash := sha256.Sum256(pervHash)
		mNode.Data = hash[:]
	}

	mNode.Left = left
	mNode.Right = right

	return &mNode
}

// NewMerkleTree 创建一颗Merkle树
func NewMerkleTree(data [][]byte) *MerkleTree {
	var nodes []MerkleNode

	if len(data)%2 != 0 {
		data = append(data, data[len(data)-1])
	}

	for _, datum := range data {
		mNode := NewMerkleNode(nil, nil, datum)
		nodes = append(nodes, *mNode)
	}

TreeNode:
	for {
		var newLevel []MerkleNode

		for j := 0; j < len(nodes); j += 2 {
			if len(nodes) == 1 {
				// 表示到了树根了
				break TreeNode
			}
			node := NewMerkleNode(&nodes[j], &nodes[j+1], nil)
			newLevel = append(newLevel, *node)
		}

		nodes = newLevel
	}

	mTree := MerkleTree{&nodes[0]}

	return &mTree

}
