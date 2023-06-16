package main

import (
	"encoding/hex"
	"fmt"
	bolt "github.com/boltdb/bolt"
	"log"
	"os"
)

// 数据库
const dbFile = "blockchain.db"
const blocksBucket = "blocks"
const genesisCoinbaseData = "The Times 16/Jun/2023 Chancellor on brink of second bailout for banks"

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

func (bc *Blockchain) Iterator() *BlockchainIterator {
	bci := &BlockchainIterator{bc.tip, bc.db}
	return bci
}

//func (bc *Blockchain) AddBlock(data string) {
//	var lastHash []byte
//
//	bc.db.View(
//		func(tx *bolt.Tx) error {
//			b := tx.Bucket([]byte(blocksBucket))
//			lastHash = b.Get([]byte("l"))
//
//			return nil
//		},
//	)
//
//	newBlock := NewBlock(data, lastHash)
//
//	bc.db.Update(func(tx *bolt.Tx) error {
//		b := tx.Bucket([]byte(blocksBucket))
//		b.Put(newBlock.Hash, newBlock.Serialize())
//		_ = b.Put([]byte("l"), newBlock.Hash)
//		bc.tip = newBlock.Hash
//
//		return nil
//	})
//}

// NewBlockchain address在这里还没用上
func NewBlockchain(address string) *Blockchain {
	if dbExists() == false {
		fmt.Println("No existing blockchain found. Create one first.")
		os.Exit(1)
	}

	var tip []byte
	//0600表示只有创建者可以修改
	db, err := bolt.Open(dbFile, 0600, nil)

	if err != nil {
		log.Panic(err)
	}
	//咋说呢，感觉 Update(fn func(*Tx) error)这样的语法，如果有嵌套行为的话很难看啊``````
	//err到处飞，不优雅
	err = db.Update(
		func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(blocksBucket))
			tip = b.Get([]byte("l"))
			return nil
		})
	if err != nil {
		log.Panic(err)
	}
	bc := Blockchain{tip, db}
	return &bc
}

// MineBlock 挖矿
func (bc *Blockchain) MineBlock(transactions []*Transaction) {
	var lastHash []byte

	err := bc.db.View(
		func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(blocksBucket))
			lastHash = b.Get([]byte("l"))

			return nil
		},
	)

	if err != nil {
		log.Panic(err)
	}

	newBlock := NewBlock(transactions, lastHash)

	err = bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		err := b.Put(newBlock.Hash, newBlock.Serialize())
		if err != nil {
			log.Panic(err)
		}

		err = b.Put([]byte("l"), newBlock.Hash)
		if err != nil {
			log.Panic(err)
		}

		bc.tip = newBlock.Hash

		return nil
	})
}

// FindUnspentTransactions 遍历所有交易，返回未交易的数据
func (bc *Blockchain) FindUnspentTransactions(address string) []Transaction {
	var unspentTXs []Transaction
	spentTXOs := make(map[string][]int)
	bci := bc.Iterator()

	for {
		//遍历每一个区块
		block := bci.Next()

		//遍历区块里的所有交易
		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

			//寻找所有输出
		Outputs:
			for outIdx, out := range tx.Vout {
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}

				if out.CanBeUnlockedWith(address) {
					unspentTXs = append(unspentTXs, *tx)
				}
			}

			//寻找所有输入
			//排除创始区块
			if tx.IsCoinbase() == false {
				for _, in := range tx.Vin {
					if in.CanUnlockOutputWith(address) {
						inTxID := hex.EncodeToString(in.Txid)
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout)
					}
				}
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return unspentTXs
}

// FindUTXO 寻找所有和address相关的未花费输出
func (bc *Blockchain) FindUTXO(address string) []TXOutput {
	var UTXOs []TXOutput
	unspentTransactions := bc.FindUnspentTransactions(address)

	for _, tx := range unspentTransactions {
		for _, out := range tx.Vout {
			if out.CanBeUnlockedWith(address) {
				UTXOs = append(UTXOs, out)
			}
		}
	}

	return UTXOs
}

// FindSpendableOutputs 当前地址，能用于支出的交易
func (bc *Blockchain) FindSpendableOutputs(address string, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	unspentTXs := bc.FindUnspentTransactions(address)
	accumulated := 0

Work:
	for _, tx := range unspentTXs {
		txID := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Vout {
			if out.CanBeUnlockedWith(address) && accumulated < amount {
				accumulated += out.Value
				unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)

				if accumulated >= amount {
					break Work
				}
			}
		}
	}

	return accumulated, unspentOutputs
}

func dbExists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}

func CreateBlockchain(address string) *Blockchain {
	if dbExists() {
		fmt.Println("Blockchain already exists.")
		os.Exit(1)
	}

	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		cbtx := NewCoinbaseTX(address, genesisCoinbaseData)
		genesis := NewGenesisBlock(cbtx)

		b, err := tx.CreateBucket([]byte(blocksBucket))
		if err != nil {
			log.Panic(err)
		}

		err = b.Put(genesis.Hash, genesis.Serialize())
		if err != nil {
			log.Panic(err)
		}

		err = b.Put([]byte("l"), genesis.Hash)
		if err != nil {
			log.Panic(err)
		}
		tip = genesis.Hash

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	bc := Blockchain{tip, db}

	return &bc
}
