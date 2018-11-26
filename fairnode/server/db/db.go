package db

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/p2p/discv5"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"time"
)

const DBNAME = "AndusChain"

type FairNodeDB struct {
	Mongo         *mgo.Session
	ActiveNodeCol *mgo.Collection
	MinerNode     *mgo.Collection
	OtprnList     *mgo.Collection
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
		ActiveNodeCol: session.DB(DBNAME).C("ActiveNode"),
		MinerNode:     session.DB(DBNAME).C("MinerNode"),
		OtprnList:     session.DB(DBNAME).C("OtprnList"),
	}
}

func (fnb *FairNodeDB) SaveActiveNode(enode string, coinbase common.Address) {
	// addr => 실제 address
	node, err := discv5.ParseNode(enode)
	if err != nil {
		fmt.Println("andus >> 노드 url 파싱에러 : ", err)
	}
	if n, _ := fnb.ActiveNodeCol.Find(bson.M{"enodeid": enode}).Count(); n > 0 {
		// andus >> active node update
		err := fnb.ActiveNodeCol.Update(bson.M{"enodeid": enode}, bson.M{"$set": bson.M{"time": time.Now()}})
		if err != nil {
			fmt.Println("andus >> Update err : ", err)
		}
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
}

func (fnb *FairNodeDB) GetActiveNodeList() []activeNode {
	var actlist []activeNode
	// andus >> 모든 activenode 리스트 전달.
	fnb.ActiveNodeCol.Find(nil).All(&actlist)
	return actlist
}

func (fnb *FairNodeDB) JobCheckActiveNode() {
	var activelist []activeNode
	now := time.Now()
	fnb.ActiveNodeCol.Find(nil).All(&activelist)
	for index := range activelist {
		if now.Sub(activelist[index].Time) >= (3 * time.Minute) {
			err := fnb.ActiveNodeCol.Remove(bson.M{"enodeid": activelist[index].EnodeId})
			if err != nil {
				fmt.Println("andus >> Remove enode err : ", err)
			}
		}
	}
}

func (fnb *FairNodeDB) CheckEnodeAndCoinbse(enodeId string, coinbase string) bool {
	// TODO : andus >> 1. Enode가 맞는지 확인 ( 조회 되지 않으면 팅김 )
	// TODO : andus >> 2. 해당하는 Enode가 이전에 보낸 코인베이스와 일치하는지
	var actnode activeNode
	err := fnb.ActiveNodeCol.Find(bson.M{"enodeid": enodeId}).One(&actnode)
	if err != nil {
		fmt.Println("andus >> CheckEnodeAndCoinbse find one err : ", err)
	}
	if actnode.EnodeId == "" {
		return false
	} else {
		if actnode.Coinbase != coinbase {
			return false
		}
	}
	return true
}

func (fnb *FairNodeDB) SaveMinerNode(otprnHash string, enode string) {
	// TODO : andus >> 실제 TCP에 접속한 채굴마이너를 저장
	if n, _ := fnb.MinerNode.Find(bson.M{"otprnhash": otprnHash}).Count(); n > 0 {
		fmt.Println("Update")
		err := fnb.MinerNode.Update(bson.M{"otprnhash": otprnHash}, bson.M{"$push": bson.M{"nodes": enode}})
		if err != nil {
			fmt.Println("andus >> MinerNodeUpdate err : ", err)
		}
	} else {
		err := fnb.MinerNode.Insert(&minerNode{Otprnhash: otprnHash, Nodes: []string{enode}})
		fmt.Println("Insert")
		if err != nil {
			fmt.Println("andus >> MinerNodeInsert err : ", err)
		}
	}
}

func (fnb *FairNodeDB) SaveOtprn(tsotprn fairtypes.TransferOtprn) {
	err := fnb.OtprnList.Insert(&saveotprn{OtprnHash: tsotprn.Hash.String(), TsOtprn: tsotprn})
	if err != nil {
		fmt.Println("andus >> saveotprn err : ", err)
	}
}
