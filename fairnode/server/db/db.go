package db

import (
	"errors"
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

var (
	MongDBConnectError = errors.New("MongoDB 접속에 문제가 있습니다")
)

// Mongodb url => mongodb://[username:password@]host1[:port1][,host2[:port2],...[,hostN[:portN]]][/[database][?options]]
func New(dbhost string, dbport string, pwd string, user string) (*FairNodeDB, error) {
	var url string

	if user != "" {
		url = fmt.Sprintf("mongodb://%s:%s@%s:%s", user, pwd, dbhost, dbport)
	} else {
		url = fmt.Sprintf("mongodb://%s:%s", dbhost, dbport)
	}

	session, err := mgo.Dial(url)
	if err != nil {
		return nil, MongDBConnectError
	}

	session.SetMode(mgo.Monotonic, true)

	return &FairNodeDB{
		Mongo:         session,
		ActiveNodeCol: session.DB(DBNAME).C("ActiveNode"),
		MinerNode:     session.DB(DBNAME).C("MinerNode"),
		OtprnList:     session.DB(DBNAME).C("OtprnList"),
	}, nil
}

func (fnb *FairNodeDB) SaveActiveNode(enode string, coinbase common.Address, clientport string) {
	// addr => 실제 address
	node, err := discv5.ParseNode(enode)
	if err != nil {
		fmt.Println("Error[DB] : 노드 url 파싱에러 : ", err)
	}

	tmp := activeNode{EnodeId: enode, Coinbase: coinbase.Hex(), Ip: node.IP.String(), Time: time.Now(), Port: clientport}

	if _, err := fnb.ActiveNodeCol.UpsertId(tmp.EnodeId, bson.M{"$set": tmp}); err != nil {
		log.Println("Error[DB] : SaveActiveNode ", err)
	}
}

func (fnb *FairNodeDB) GetActiveNodeNum() int {

	num, err := fnb.ActiveNodeCol.Find(nil).Count()
	if err != nil {
		log.Println("Error[DB] : GetActiveNodeNum err : ", err)
	}
	// TODO : andus >> DB에서 Active node 갯수 조회
	log.Println("Info[DB] :Db.GetActiveNodeNum -> ", num)

	return num
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
			err := fnb.ActiveNodeCol.RemoveId(activelist[index].EnodeId)
			if err != nil {
				log.Println("Error[DB] : Remove enode err : ", err)
			}
		}
	}
}

func (fnb *FairNodeDB) CheckEnodeAndCoinbse(enodeId string, coinbase string) bool {
	// TODO : andus >> 1. Enode가 맞는지 확인 ( 조회 되지 않으면 팅김 )
	// TODO : andus >> 2. 해당하는 Enode가 이전에 보낸 코인베이스와 일치하는지
	var actnode activeNode
	err := fnb.ActiveNodeCol.FindId(enodeId).One(&actnode)
	if err != nil {
		log.Println("Error[DB] : CheckEnodeAndCoinbse find one err : ", err)
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
	m := minerNode{Otprnhash: otprnHash, Nodes: []string{enode}, Timestamp: time.Now()}
	_, err := fnb.MinerNode.UpsertId(m.Otprnhash, bson.M{"$push": bson.M{"nodes": enode}, "$set": bson.M{"timestamp": m.Timestamp}})
	if err != nil {
		log.Println("Error[DB] : MinerNodeInsert err : ", err)
	}
}

func (fnb *FairNodeDB) SaveOtprn(tsotprn fairtypes.TransferOtprn) {
	err := fnb.OtprnList.Insert(&saveotprn{OtprnHash: tsotprn.Hash.String(), TsOtprn: tsotprn})
	if err != nil {
		log.Println("Error[DB] : saveotprn err : ", err)
	}
}

func (fnb *FairNodeDB) GetMinerNode(otprnHash string) []string {
	var minerlist minerNode
	err := fnb.MinerNode.FindId(otprnHash).One(&minerlist)
	if err != nil {
		log.Println("Error[DB] : GetMinerNode", err)
	}

	return minerlist.Nodes
}
