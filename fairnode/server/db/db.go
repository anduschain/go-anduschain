package db

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/p2p/discv5"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"time"
)

type FairNodeDB struct {
	Mongo         *mgo.Session
	ActiveNodeCol *mgo.Collection
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
		Mongo:         session,
		ActiveNodeCol: session.DB("AndusChain").C("ActiveNode"),
	}
}

func (fnb *FairNodeDB) SaveActiveNode(enode string, coinbase common.Address) {
	// addr => 실제 address
	node, err := discv5.ParseNode(enode)
	if err != nil {
		fmt.Println("andus >> 노드 url 파싱에러 : ", err)
	}
	if n, _ := fnb.ActiveNodeCol.Find(bson.M{"enodeid": enode}).Count(); n > 0 {
		return
	} else {
		err = fnb.ActiveNodeCol.Insert(&activeNode{EnodeId: enode, Coinbase: coinbase.Hex(), Ip: node.IP.String(), Time: time.Now()})
		if err != nil {
			fmt.Println("andus >> SaveActiveNode error : ", err)
		}
	}
}

func (fnb *FairNodeDB) GetActiveNodeNum() int {

	num, err := fnb.ActiveNodeCol.Find(nil).Count()
	if err != nil {
		fmt.Println("andus >> GetActiveNodeNum err : ", err)
	}
	// TODO : andus >> DB에서 Active node 갯수 조회
	log.Println("andus >> Db.GetActiveNodeNum : ", num)

	//return num
	return 3
	// TODO : test 필요
}

func (fnb *FairNodeDB) GetActiveNodeList() []activeNode {

	// TODO : andus >> DB에서 Active node 리스트를 조회
	log.Println("andus >> Db.GetActiveNodeList")

	var enodeid []activeNode
	//return example //[]string{"121.134.35.45:50002"}
	var test []string

	// TODO : andus >> mongodb에서 enode만 가져오기
	fnb.ActiveNodeCol.Find(nil).All(&enodeid)
	fnb.ActiveNodeCol.Find(nil).Select(&enodeid)
	fmt.Println("엑티브노드들 enodeid : ", test)
	return enodeid
}

func (fnb *FairNodeDB) JobCheckActiveNode() {
	// TODO : Active Node 관리 (주기 : 3분)..

	fmt.Println(fnb.ActiveNodeCol.Find(bson.M{"time": true}))

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
