package db

// TODO : db status check 변수
// TODO : db CRUD interface

type DB interface {
	// TODO :  DB interface
	Create(val interface{}) bool
	Select(val interface{}) ([]byte, error)
	Update(val interface{}) (bool, int)
	Delete(val interface{}) (bool, int)
	Insert(val interface{}) (bool, int)
}

type FairNodeDB struct {
}

func New() (*FairNodeDB, error) {
	// TODO : mongodb 연결 및 사용정보...

	return &FairNodeDB{}, nil
}

func (db *FairNodeDB) Create(val interface{}) bool {

	return false
}

func (db *FairNodeDB) Select(val interface{}) ([]byte, error) {

	return []byte{}, nil
}

func (db *FairNodeDB) Update(val interface{}) (bool, int) {

	return false, 0
}

func (db *FairNodeDB) Insert(val interface{}) (bool, int) {

	return false, 0
}

func JobCheckActiveNode() error {
	// TODO : Active Node 관리 (주기 : 3분)..

	return nil
}
