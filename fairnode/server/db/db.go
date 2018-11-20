package db

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/p2p/discv5"
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

func (fnb *FairNodeDB) SaveActiveNode(enode discv5.Node, addr *net.UDPAddr, coinbase common.Address) bool {

	// addr => 실제 address
	activenodeCol := fnb.Mongo.DB("AndusChain").C("ActiveNode")
	activenodeCol.Insert(&activeNode{EnodeId: enode.ID.String(), Coinbase: coinbase.Hex(), Ip: addr.IP.String(), Time: time.Now().UnixNano()})

	log.Println("andus >> DB에 insert Or Update 호출")

	return false
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
