package db

// TODO : db status check 변수
// TODO : db CRUD interface

type DB interface {
	// TODO :  DB interface
}

type FairNodeDB struct {
}

func New() (*FairNodeDB, error) {
	// TODO : mongodb 연결 및 사용정보...

	return &FairNodeDB{}, nil
}

func JobCheckActiveNode() error {
	// TODO : Active Node 관리 (주기 : 3분)..

	return nil
}
