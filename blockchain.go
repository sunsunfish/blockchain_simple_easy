package main

import (
	bolt "github.com/boltdb/bolt"
)

// 数据库
const dbFile = "blockchain.db"
const blocksBucket = "blocks"

type Blockchain struct {
	//最后一个区块
	tip []byte

	db *bolt.DB
}

type BlockchainIterator struct {
	currentHash []byte
	db          *bolt.DB
}

func (i *BlockchainIterator) Next() *Block {
	var block *Block
	i.db.View(
		func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(blocksBucket))
			encodeBlock := b.Get(i.currentHash)
			block = DeserializeBlock(encodeBlock)
			return nil
		},
	)
	i.currentHash = block.PrevBlockHash
	return block
}

func (b *Blockchain) Iterator() *BlockchainIterator {
	bci := &BlockchainIterator{b.tip, b.db}
	return bci
}

func (bc *Blockchain) AddBlock(data string) {
	var lastHash []byte

	bc.db.View(
		func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(blocksBucket))
			lastHash = b.Get([]byte("l"))

			return nil
		},
	)

	newBlock := NewBlock(data, lastHash)

	bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		b.Put(newBlock.Hash, newBlock.Serialize())
		_ = b.Put([]byte("l"), newBlock.Hash)
		bc.tip = newBlock.Hash

		return nil
	})
}

func NewBlockchain() *Blockchain {
	var tip []byte
	//0600表示只有创建者可以修改
	db, _ := bolt.Open(dbFile, 0600, nil)

	//咋说呢，感觉 Update(fn func(*Tx) error)这样的语法，如果有嵌套行为的话很难看啊``````
	//err到处飞，不优雅
	db.Update(
		func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(blocksBucket))
			if b == nil {
				genesis := NewGenesisBlock()
				b, _ := tx.CreateBucket([]byte(blocksBucket))
				_ = b.Put(genesis.Hash, genesis.Serialize())
				//l键永远是最后一个区块
				_ = b.Put([]byte("l"), genesis.Hash)
				tip = genesis.Hash
			} else {
				tip = b.Get([]byte("l"))
			}
			return nil
		},
	)
	bc := Blockchain{tip, db}
	return &bc
}
