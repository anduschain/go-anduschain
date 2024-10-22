package ordererdb

import proto "github.com/anduschain/go-anduschain/protos/common"

type OrdererDB interface {
	Start() error
	Stop()
	InsertTransactionToTxPool(sender string, nonce uint64, hash string, tx []byte) error
	GetTransactionListFromTxPool() ([]proto.Transaction, error)
}
