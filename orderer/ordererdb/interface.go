package ordererdb

type OrdererDB interface {
	Start() error
	Stop()
}
