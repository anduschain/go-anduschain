package ordererdb

type OrdererDB interface {
	Start() error
	Stop()
	//InsertTransactionsToTxPool(transactions []*proto.Transaction) error
}
