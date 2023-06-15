package main

import (
	"bytes"
	"crypto/sha256"
	"strconv"
	"time"
)

// Block 区块结构体
type Block struct {
	//创建区块时的时间戳
	Timestamp int64

	Data []byte

	PrevBlockHash []byte

	//当前区块的hash，也是下一个区块的PrevBlockHash
	Hash []byte

	Nonce int
}

func (b *Block) SetHash() {
	//把时间戳转化为10进制字符串，然后再转化为字节数组
	//用不同进制转化时间戳，字符长度会不一样，在需要高效率的哈希计算场景下，较短的字符串可以减少计算时间。而在需要更直观可读的场景下，则可以使用较长的字符串
	timestamp := []byte(strconv.FormatInt(b.Timestamp, 10))

	//将这三个部分的字节数组通过空字符连接起来，形成一个新的字节数组作为最终的哈希输入
	headers := bytes.Join([][]byte{b.PrevBlockHash, b.Data, timestamp}, []byte{})
	hash := sha256.Sum224(headers)
	b.Hash = hash[:]
}

func NewBlock(data string, prevlockHash []byte) *Block {
	block := &Block{time.Now().Unix(), []byte(data), prevlockHash, []byte{}, 0}
	pow := NewProofWork(block)
	nonce, hash := pow.Run()
	block.Hash = hash[:]
	block.Nonce = nonce
	return block
}

func NewGenesisBlock() *Block {
	return NewBlock("Genesis Block", []byte{})
}
