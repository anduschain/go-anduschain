package db

// TODO : db status check 변수
// TODO : db CRUD interface

type DB interface {
	// TODO :  DB interface
}

type FairNodeDB struct {
}

func New() (*FairNodeDB, error) {
	// TODO : mongodb 연결 및 사용 준비...

	return &FairNodeDB{}, nil
}
