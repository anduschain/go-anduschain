package db

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"gopkg.in/mgo.v2"
	"log"
	"net"
	"time"
)

type FairNodeDB struct {
	Mongo *mgo.Session
}

// Mongodb url => mongodb://[username:password@]host1[:port1][,host2[:port2],...[,hostN[:portN]]][/[database][?options]]
func New(dbhost string, dbport string, pwd string) *FairNodeDB {
	// TODO : mongodb 연결 및 사용정보...
	// mongodb://username:pwd@localhost:3000
	//username := "deb"
	//url := fmt.Sprintf("mongodb://%s:%s@%s:%s", username, pwd, dbhost, dbport)
	url := "mongodb://localhost:27017"
	session, err := mgo.Dial(url)
	if err != nil {
		log.Fatal("andus >> MongoDB 접속에 문제가 있습니다")
	}
	session.SetMode(mgo.Monotonic, true)

	return &FairNodeDB{
		Mongo: session,
	}
}

func (fnb *FairNodeDB) SaveActiveNode(enode string, addr *net.UDPAddr, coinbase common.Address) {

	// addr => 실제 address
	activenodeCol := fnb.Mongo.DB("AndusChain").C("ActiveNode")
	activenodeCol.Insert(&activeNode{EnodeId: enode, Coinbase: coinbase.Hex(), Ip: addr.IP.String(), Time: time.Now()})

	log.Println("andus >> DB에 insert Or Update 호출")
}

func (fnb *FairNodeDB) GetActiveNodeNum() int {

	// TODO : andus >> DB에서 Active node 갯수 조회
	log.Println("andus >> Db.GetActiveNodeNum")

	return 3
}

func (fnb *FairNodeDB) GetActiveNodeList() []string {

	// TODO : andus >> DB에서 Active node 리스트를 조회
	log.Println("andus >> Db.GetActiveNodeList")

	//return example //[]string{"121.134.35.45:50002"}

	return []string{"121.134.35.45:50002", "121.134.35.45:50003"}
}

func (fnb *FairNodeDB) JobCheckActiveNode() {
	// TODO : Active Node 관리 (주기 : 3분)..
}

func (fnb *FairNodeDB) CheckEnodeAndCoinbse(enodeId string, coinbase string) bool {
	// TODO : andus >> 1. Enode가 맞는지 확인 ( 조회 되지 않으면 팅김 )
	// TODO : andus >> 2. 해당하는 Enode가 이전에 보낸 코인베이스와 일치하는지

	fmt.Println("andus >> CheckEnodeAndCoinbse call")
	return true
}

func (fnb *FairNodeDB) SaveMinerNode(otprnNum uint64, enode string) {
	// TODO : andus >> 실제 TCP에 접속한 채굴마이너를 저장
}
