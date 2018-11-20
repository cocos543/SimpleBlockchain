package main

import (
	"bytes"
	"encoding/gob"
	"time"
)

type Block struct {
	Timestamp     int64
	Transactions  []*Transaction
	PrevBlockHash []byte
	Hash          []byte
	Nonce         int

	// 用于临时保存计算出的交易hash
	txMerkleTreeRootHash []byte
}

// Serialize 序列化Block结构体
func (b *Block) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)

	err := encoder.Encode(b)
	if err != nil {
		panic(err)
	}

	return result.Bytes()
}

// DeserializeBlock 反序列化, 这是一个函数不是方法
func DeserializeBlock(d []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	if err != nil {
		panic(err)
	}

	return &block
}

func NewBlock(transactions []*Transaction, prevBlockHash []byte) *Block {
	block := &Block{time.Now().Unix(), transactions, prevBlockHash, []byte{}, 0, nil}
	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

func NewGenesisBlock(coinbase *Transaction) *Block {
	return NewBlock([]*Transaction{coinbase}, []byte{})
}

func (b *Block) HashTransactions() []byte {
	// 这样可以避免挖矿过程中重复计算出同一个MerkleTree rootHash
	if b.txMerkleTreeRootHash != nil {
		return b.txMerkleTreeRootHash
	}
	var transactions [][]byte

	for _, tx := range b.Transactions {
		transactions = append(transactions, tx.Serialize())
	}

	mTree := NewMerkleTree(transactions)
	b.txMerkleTreeRootHash = mTree.RootNode.Data

	return mTree.RootNode.Data
}
