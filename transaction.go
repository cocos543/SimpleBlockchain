package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"
)

const subsidy = 50

type Transaction struct {
	ID   []byte
	Vin  []TXInput
	Vout []TXOutput
}

type TXOutput struct {
	Value int
	//ScriptPubKey string
	PubKeyHash []byte
}

// NewTXOutput create a new TXOutput
func NewTXOutput(value int, address string) *TXOutput {
	txo := &TXOutput{value, nil}
	txo.Lock([]byte(address))

	return txo
}

func (out *TXOutput) Lock(address []byte) {
	pubKeyHash := HashPubKeyFromAddress(address)
	out.PubKeyHash = pubKeyHash
}

func (out *TXOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(out.PubKeyHash, pubKeyHash) == 0
}

// String returns a human-readable representation of a TXOutput
func (output TXOutput) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("\n--- TXOutput"))

	lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
	lines = append(lines, fmt.Sprintf("       PubKeyHash: %x", output.PubKeyHash))

	return strings.Join(lines, "\n")
}

// TXOutputs collects TXOutput
type TXOutputs struct {
	Outputs []TXOutput
}

// Serialize serializes TXOutputs
func (outs TXOutputs) Serialize() []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(outs)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

// DeserializeOutputs deserializes TXOutputs
func DeserializeOutputs(data []byte) TXOutputs {
	var outputs TXOutputs

	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&outputs)
	if err != nil {
		log.Panic(err)
	}

	return outputs
}

//////////////////////////////////////下面是输入

type TXInput struct {
	Txid []byte
	//引用了Txid这个交易的输出的索引
	Vout int
	//ScriptSig string
	Signature []byte
	// input保存的公钥, 这是因为签名验证的时候, 需要拿出该公钥进行验证; output保存的是公钥的哈希, 这是因为输出不需要公钥做验证
	PubKey []byte
}

func (in *TXInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := HashPubKey(in.PubKey)

	return bytes.Compare(lockingHash, pubKeyHash) == 0
}

// IsCoinbase checks whether the transaction is coinbase
func (tx Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

// Serialize 序列化Block结构体
func (tx *Transaction) Serialize() []byte {
	var buff bytes.Buffer
	encoder := gob.NewEncoder(&buff)

	err := encoder.Encode(tx)
	if err != nil {
		panic(err)
	}

	return buff.Bytes()
}

// DeserializeTransaction deserializes a transaction
func DeserializeTransaction(data []byte) Transaction {
	var transaction Transaction

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&transaction)
	if err != nil {
		log.Panic(err)
	}

	return transaction
}

// Hash returns the hash of the Transaction
func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	log.Printf("txCopy:%s\n", txCopy)
	log.Printf("txCopy.Serialize()=>%x", txCopy.Serialize())
	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

// func (in *TXInput) CanUnlockOutputWith(unlockingData string) bool {
// 	return in.ScriptSig == unlockingData
// }

// func (out *TXOutput) CanBeUnlockedWith(unlockingData string) bool {
// 	return out.ScriptPubKey == unlockingData
// }

func NewUTXOTransaction(wallet *Wallet, from, to string, amount int, UTXOSet *UTXOSet) *Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	pubKeyHash := HashPubKey(wallet.PublicKey)
	acc, validOutputs := UTXOSet.FindSpendableOutputs(pubKeyHash, amount)

	log.Printf("\nvalidOutputs:%#v\n\n", validOutputs)
	if acc < amount {
		log.Panic("ERROR: Not enough funds")
	}

	// Build a list of inputs
	for txid, outs := range validOutputs {
		txID, _ := hex.DecodeString(txid)

		for _, out := range outs {
			// 这里的out其实就是某个txID这个交易里面的Vout数组的索引
			// input保存的公钥, 这是因为签名验证的时候, 需要拿出该公钥进行验证; output保存的是公钥的哈希, 这是因为输出不需要公钥做验证
			input := TXInput{txID, out, nil, wallet.PublicKey}
			inputs = append(inputs, input)
		}
	}

	// Build a list of outputs
	// 这里虽然有2个output, 但是每一个地址最多只有一个output
	outputs = append(outputs, *NewTXOutput(amount, to))
	if acc > amount {
		// 这里就是找零.
		outputs = append(outputs, *NewTXOutput(acc-amount, from)) // a change
	}

	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()
	UTXOSet.Blockchain.SignTransaction(&tx, wallet.PrivateKey)
	log.Printf("\nnewTx:%s\n\n", tx)
	return &tx
}

func NewCoinbaseTX(to, data string) *Transaction {
	if data == "" {
		randData := make([]byte, 20)
		_, err := rand.Read(randData)
		if err != nil {
			log.Panic(err)
		}

		data = fmt.Sprintf("%x", randData)
	}

	txin := TXInput{[]byte{}, -1, nil, []byte(data)}
	txout := *NewTXOutput(subsidy, to)
	tx := Transaction{nil, []TXInput{txin}, []TXOutput{txout}}
	tx.ID = tx.Hash()
	log.Printf("\nnewCoinbaseTx:%s\n\n", tx)
	return &tx
}

// String returns a human-readable representation of a transaction
func (tx Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("\n--- Transaction %x:", tx.ID))

	for i, input := range tx.Vin {

		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TXID:      %x", input.Txid))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.Vout))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))
	}

	for i, output := range tx.Vout {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       PubKeyHash: %x", output.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}

// Sign 对交易签名, 签名之后的数据会放在input的Signature字段里面
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey, prevTXs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	}

	txCopy := tx.TrimmedCopy()

	for inID, vin := range txCopy.Vin {
		// 从上一个块的交易列表中选出当前输入引用的那些tx
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		// 把输入引用的"上一个交易输出的公钥哈希", 赋值过来
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash
		// 上面那些步骤就是为了计算出hash, 用来签名

		//参考验证部分的注释
		//txCopy.ID = txCopy.Hash()

		txCopy.Vin[inID].PubKey = nil
		dataToSign := fmt.Sprintf("%x", txCopy)
		log.Printf("Sign Data:0x%s\n\n", dataToSign)

		r, s, err := ecdsa.Sign(rand.Reader, &privKey, []byte(dataToSign))
		if err != nil {
			panic(err)
		}
		signature := append(r.Bytes(), s.Bytes()...)

		tx.Vin[inID].Signature = signature
	}
}

// TrimmedCopy 从交易中拷贝出需要用到的信息创建一个新的实例
func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	for _, vin := range tx.Vin {
		inputs = append(inputs, TXInput{vin.Txid, vin.Vout, nil, nil})
	}

	for _, vout := range tx.Vout {
		outputs = append(outputs, TXOutput{vout.Value, vout.PubKeyHash})
	}

	txCopy := Transaction{tx.ID, inputs, outputs}

	return txCopy
}

// Verify 验证每一个tx里面的签名
func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	// Coinbase 为矿工奖励, 所以不需要验证
	if tx.IsCoinbase() {
		return true
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inID, vin := range tx.Vin {
		// 这里和签名的代码一样
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash

		//这里原先是用Hash作为签名数据, 但是从钱包节点->中心节点传过来的tx虽然字段值都一样, 但是序列化的字节却有一些不同, 导致最终hash出来的结果不同
		//所以现在修改成直接把结构体打印成十六进制dataToSign再用于签名
		//txCopy.ID = txCopy.Hash()

		txCopy.Vin[inID].PubKey = nil

		r := big.Int{}
		s := big.Int{}
		sigLen := len(vin.Signature)
		r.SetBytes(vin.Signature[:(sigLen / 2)])
		s.SetBytes(vin.Signature[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}
		keyLen := len(vin.PubKey)
		x.SetBytes(vin.PubKey[:(keyLen / 2)])
		y.SetBytes(vin.PubKey[(keyLen / 2):])

		rawPubKey := ecdsa.PublicKey{Curve: curve, X: &x, Y: &y}

		dataToSign := fmt.Sprintf("%x", txCopy)
		log.Printf("Verify Data:0x%x\n", dataToSign)
		if ecdsa.Verify(&rawPubKey, []byte(dataToSign), &r, &s) == false {
			return false
		}
	}

	return true
}
