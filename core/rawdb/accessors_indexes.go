// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rawdb

import (
	"bytes"
	"math/big"

	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/ethdb"
	"github.com/anduschain/go-anduschain/log"
	"github.com/anduschain/go-anduschain/params"
	"github.com/anduschain/go-anduschain/rlp"
)

// ReadTxLookupEntry retrieves the positional metadata associated with a transaction
// hash to allow retrieving the transaction or receipt by hash.
func ReadTxLookupEntry(db ethdb.Reader, hash common.Hash) (common.Hash, uint64, uint64) {
	data, _ := db.Get(txLookupKey(hash))
	if len(data) == 0 {
		return common.Hash{}, 0, 0
	}
	var entry LegacyTxLookupEntry
	if err := rlp.DecodeBytes(data, &entry); err != nil {
		log.Error("Invalid transaction lookup entry RLP", "hash", hash, "err", err)
		return common.Hash{}, 0, 0
	}
	return entry.BlockHash, entry.BlockIndex, entry.Index
}

// writeTxLookupEntry stores a positional metadata for a transaction,
// enabling hash based transaction and receipt lookups.
func writeTxLookupEntry(db ethdb.KeyValueWriter, hash common.Hash, numberBytes []byte) {
	if err := db.Put(txLookupKey(hash), numberBytes); err != nil {
		log.Crit("Failed to store transaction lookup entry", "err", err)
	}
}

// WriteTxLookupEntries is identical to WriteTxLookupEntry, but it works on
// a list of hashes
func WriteTxLookupEntries(db ethdb.KeyValueWriter, number uint64, hashes []common.Hash) {
	numberBytes := new(big.Int).SetUint64(number).Bytes()
	for _, hash := range hashes {
		writeTxLookupEntry(db, hash, numberBytes)
	}
}

// WriteTxLookupEntriesByBlock stores a positional metadata for every transaction from
// a block, enabling hash based transaction and receipt lookups.
func WriteTxLookupEntriesByBlock(db ethdb.KeyValueWriter, block *types.Block) {
	for i, tx := range block.Transactions() {
		entry := LegacyTxLookupEntry{
			BlockHash:  block.Hash(),
			BlockIndex: block.NumberU64(),
			Index:      uint64(i),
		}
		data, err := rlp.EncodeToBytes(entry)
		if err != nil {
			log.Crit("Failed to encode transaction lookup entry", "err", err)
		}
		if err := db.Put(txLookupKey(tx.Hash()), data); err != nil {
			log.Crit("Failed to store transaction lookup entry", "err", err)
		}
	}
}

// DeleteTxLookupEntry removes all transaction data associated with a hash.
func DeleteTxLookupEntry(db ethdb.KeyValueWriter, hash common.Hash) {
	if err := db.Delete(txLookupKey(hash)); err != nil {
		log.Crit("Failed to delete transaction lookup entry", "err", err)
	}
}

// DeleteTxLookupEntries removes all transaction lookups for a given block.
func DeleteTxLookupEntries(db ethdb.KeyValueWriter, hashes []common.Hash) {
	for _, hash := range hashes {
		DeleteTxLookupEntry(db, hash)
	}
}

// ReadTransaction retrieves a specific transaction from the database, along with
// its added positional metadata.
func ReadTransaction(db ethdb.Reader, hash common.Hash) (*types.Transaction, common.Hash, uint64, uint64) {
	_, blockNumber, _ := ReadTxLookupEntry(db, hash)
	if blockNumber == 0 {
		return nil, common.Hash{}, 0, 0
	}
	blockHash := ReadCanonicalHash(db, blockNumber)
	if blockHash == (common.Hash{}) {
		return nil, common.Hash{}, 0, 0
	}
	body := ReadBody(db, blockHash, blockNumber)
	if body == nil {
		log.Error("Transaction referenced missing", "number", blockNumber, "hash", blockHash)
		return nil, common.Hash{}, 0, 0
	}
	for txIndex, tx := range body.Transactions {
		if tx.Hash() == hash {
			return tx, blockHash, blockNumber, uint64(txIndex)
		}
	}
	log.Error("Transaction not found", "number", blockNumber, "hash", blockHash, "txhash", hash)
	return nil, common.Hash{}, 0, 0
}

// ReadReceipt retrieves a specific transaction receipt from the database, along with
// its added positional metadata.
func ReadReceipt(db ethdb.Reader, hash common.Hash, config *params.ChainConfig) (*types.Receipt, common.Hash, uint64, uint64) {
	// Retrieve the context of the receipt based on the transaction hash
	_, blockNumber, _ := ReadTxLookupEntry(db, hash)
	if blockNumber == 0 {
		return nil, common.Hash{}, 0, 0
	}
	blockHash := ReadCanonicalHash(db, blockNumber)
	if blockHash == (common.Hash{}) {
		return nil, common.Hash{}, 0, 0
	}
	// Read all the receipts from the block and return the one with the matching hash
	receipts := ReadReceipts(db, blockHash, blockNumber, config)
	for receiptIndex, receipt := range receipts {
		if receipt.TxHash == hash {
			return receipt, blockHash, blockNumber, uint64(receiptIndex)
		}
	}
	log.Error("Receipt not found", "number", blockNumber, "hash", blockHash, "txhash", hash)
	return nil, common.Hash{}, 0, 0
}

// ReadBloomBits retrieves the compressed bloom bit vector belonging to the given
// section and bit index from the.
func ReadBloomBits(db ethdb.KeyValueReader, bit uint, section uint64, head common.Hash) ([]byte, error) {
	return db.Get(bloomBitsKey(bit, section, head))
}

// WriteBloomBits stores the compressed bloom bits vector belonging to the given
// section and bit index.
func WriteBloomBits(db ethdb.KeyValueWriter, bit uint, section uint64, head common.Hash, bits []byte) {
	if err := db.Put(bloomBitsKey(bit, section, head), bits); err != nil {
		log.Crit("Failed to store bloom bits", "err", err)
	}
}

// DeleteBloombits removes all compressed bloom bits vector belonging to the
// given section range and bit index.
func DeleteBloombits(db ethdb.Database, bit uint, from uint64, to uint64) {
	start, end := bloomBitsKey(bit, from, common.Hash{}), bloomBitsKey(bit, to, common.Hash{})
	it := db.NewIterator(nil, start)
	defer it.Release()

	for it.Next() {
		if bytes.Compare(it.Key(), end) >= 0 {
			break
		}
		if len(it.Key()) != len(bloomBitsPrefix)+2+8+32 {
			continue
		}
		db.Delete(it.Key())
	}
	if it.Error() != nil {
		log.Crit("Failed to delete bloom bits", "err", it.Error())
	}
}
