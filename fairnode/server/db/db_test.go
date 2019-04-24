package db

import (
	"fmt"
	"gopkg.in/mgo.v2/bson"
	"log"
	"strings"
	"testing"
	"time"
)

var session, _ = New(nil)

type enodeid2 struct {
	Enodeid string
}

func TestFairNodeDB_SaveActiveNode(t *testing.T) {
	p := []activeNode{
		{EnodeId: "testenode1", Ip: "10101010", Coinbase: "0X82af8219dfe09f18d8ebf8075e31b81a01a8c4fd8b9a98f60a258e0aa1a8da", Time: time.Now()},
		{EnodeId: "testenode1", Ip: "10101010", Coinbase: "0X82af8219dfe09f18d8ebf8075e31b81a01a8c4fd8b9a98f60a258e0aa1a8da", Time: time.Now()},
		{EnodeId: "testenode2", Ip: "10101010", Coinbase: "0X82af8219dfe09f18d8ebf8075e31b81a01a8c4fd8b9a98f60a258e0aa1a8da", Time: time.Now()},
		{EnodeId: "testenode3", Ip: "10101010", Coinbase: "0X82af8219dfe09f18d8ebf8075e31b81a01a8c4fd8b9a98f60a258e0aa1a8da", Time: time.Now()},
		{EnodeId: "testenode4", Ip: "10101010", Coinbase: "0X82af8219dfe09f18d8ebf8075e31b81a01a8c4fd8b9a98f60a258e0aa1a8da", Time: time.Now()},
	}

	for index := range p {

		Info, err := session.ActiveNodeCol.UpsertId(p[index].EnodeId, bson.M{"$set": p[index]})
		if err != nil {
			log.Println("Error : MinerNodeInsert err : ", err)
		}
		log.Println("Debug[andus]", Info)
	}
	var tmp []activeNode
	session.ActiveNodeCol.Find(nil).All(&tmp)
	fmt.Println(tmp[0].EnodeId)

}

func SaveMinerNode2(otprnHash string, enode string) {

	var p = struct {
		OtprnHash string
		Nodes     []string
	}{otprnHash, []string{enode}}

	Info, err := session.MinerNode.UpsertId(p.OtprnHash, bson.M{"$push": bson.M{"nodes": enode}})
	if err != nil {
		log.Println("Error : MinerNodeInsert err : ", err)
	}

	log.Println("Debug[andus]", Info)
}

func TestFairNodeDB_GetActiveNodeList(t *testing.T) {
	var result []activeNode

	var test2 []enodeid2
	session.ActiveNodeCol.Find(nil).All(&result)
	session.ActiveNodeCol.Find(nil).All(&test2)
	fmt.Println("1번 결과 result: ", result)
	fmt.Println("2번 결과 test  : ", test2)
	fmt.Println("ddd", test2[0])
	defer session.Mongo.Close()
}

func TestFairNodeDB_SaveOtprn(t *testing.T) {
	var save saveotprn
	session.OtprnList.Find(bson.M{"otprnhash": "0x82af8219dfe09f18d8ebf8075e31b81a01a8c4fd8b9a98f60a258e0aa1a8daf0"}).One(&save)

	fmt.Println(save.TsOtprn.Otp.HashOtprn().String() == "0x82af8219dfe09f18d8ebf8075e31b81a01a8c4fd8b9a98f60a258e0aa1a8daf0")
	fmt.Println(save.TsOtprn.Hash == save.TsOtprn.Otp.HashOtprn())

	defer session.Mongo.Close()
}

func TestFairNodeDB_SaveMinerNode(t *testing.T) {
	defer session.Mongo.Close()

	testList := []struct {
		otprnHash string
		enode     string
	}{
		{"0x82af8219dfe09f18d8ebf8075e31b81a01a8c4fd8b9a98f60a258e0aa1a8daf0",
			"enode://152a309b1c09bf3989295062e379c0f28e7ae814aed34f291f3c856f46aadde3e20d4c08f6e238b9f65b242b08b76c88bdaf97ecf386e869eba24711adca0c1f@121.134.35.45:30303",
		},
		{"0x82af8219dfe09f18d8ebf8075e31b81a01a8c4fd8b9a98f60a258e0aa1a8daf0",
			"enode://152a309b1c09bf3989295062e379c0f28e7ae814aed34f291f3c856f46aadde3e20d4c08f6e238b9f65b242b08b76c88bdaf97ecf386e869eba24711adca0c1f@121.134.35.45:30303",
		},
		{"0x82af8219dfe09f18d8ebf8075e31b81a01a8c4fd8b9a98f60a258e0aa1a8daf0",
			"enode://152a309b1c09bf3989295062e379c0f28e7ae814aed34f291f3c856f46aadde3e20d4c08f6e238b9f65b242b08b76c88bdaf97ecf386e869eba24711adca0c1f@121.134.35.45:30303",
		},
		{"0x82af8219dfe09f18d8ebf8075e31b81a01a8c4fd8b9a98f60a258e0aa1a8daf0",
			"enode://152a309b1c09bf3989295062e379c0f28e7ae814aed34f291f3c856f46aadde3e20d4c08f6e238b9f65b242b08b76c88bdaf97ecf386e869eba24711adca0c1f@121.134.35.45:30303",
		},
		{"0x82af8219dfe09f18d8ebf8075e31b81a01a8c4fd8b9a98f60a258e0aa1a8daf0",
			"enode://152a309b1c09bf3989295062e379c0f28e7ae814aed34f291f3c856f46aadde3e20d4c08f6e238b9f65b242b08b76c88bdaf97ecf386e869eba24711adca0c1f@121.134.35.45:30303",
		},
	}

	for i := range testList {
		session.SaveMinerNode(testList[i].otprnHash, testList[i].enode)
	}
}

func TestFairNodeDB_JobCheckActiveNode(t *testing.T) {

}

func TestFairNodeDB_GetCurrentBlock(t *testing.T) {
	defer session.Stop()
	session.Start()

	qury := func() int64 {
		var count *storedBlock
		err := session.BlockChain.Find(bson.M{}).Sort("-header.number").Limit(1).One(&count)
		if err != nil {
			log.Println("Error[DB] : GetCurrentBlock", err)
		}

		if count == nil {
			return 0
		}

		return count.Header.Number
	}

	for i := 0; i < 10; i++ {
		fmt.Println("Currnet block", qury())
	}
}

func TestFairNodeDB_GetRawBlock(t *testing.T) {
	defer session.Stop()
	session.Start()

	hashs := []struct {
		hash    string
		txCount int64
	}{
		{"0xe4c90f3174ee8c98728f80c2f130f596af66a656383a9b52714e4301d91b467e", 15},
		{"0x59e6e1097b172bc2f281c953d76f13bb76be6b79d7ca4fd18ebcf7af34f4952e", 3},
	}

	for i := 0; i < len(hashs); i++ {
		block, err := session.GetRawBlock(hashs[i].hash)
		if err != nil {
			t.Error(err)
		}

		if strings.Compare(block.Hash().String(), hashs[i].hash) == 0 {
			if block.Transactions().Len() == int(hashs[i].txCount) {
				t.Log("블록이 성공적으로 가져와짐", "txcount", hashs[i].txCount, true)
				for i := range block.Transactions() {
					t.Log("txhash", block.Transactions()[i].Hash().String())
				}
			}
		} else {
			t.Log("블록을 정상적으로 가져오지 못함", false)
		}
	}

}
