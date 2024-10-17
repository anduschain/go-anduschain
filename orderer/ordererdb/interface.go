package ordererdb

type OrdererDB interface {
	Start() error
	Stop()
	InsertTransactionToTxPool(sender string, nonce uint64, hash string, tx []byte) error
}
