package db

import (
	"fmt"
	"gopkg.in/mgo.v2"
	"log"
)

type FairNodeDB struct {
	db *mgo.Session
}

// Mongodb url => mongodb://[username:password@]host1[:port1][,host2[:port2],...[,hostN[:portN]]][/[database][?options]]
func New(dbhost string, dbport string, pwd string) *FairNodeDB {
	// TODO : mongodb 연결 및 사용정보...
	// mongodb://username:pwd@localhost:3000
	username := "deb"
	url := fmt.Sprintf("mongodb://%s:%s@%s:%s", username, pwd, dbhost, dbport)
	session, err := mgo.Dial(url)
	if err != nil {
		log.Fatal("andus >> MongoDB 접속에 문제가 있습니다")
	}
	defer session.Close()

	return &FairNodeDB{
		db: session,
	}
}

func (fnb *FairNodeDB) SaveActiveNode() bool {

	log.Println("andus >> DB에 insert Or Update 호출")

	return false
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
