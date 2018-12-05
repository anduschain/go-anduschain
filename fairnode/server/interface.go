package server

type ServiceFunc interface {
	Start() error
	Stop() error
}
