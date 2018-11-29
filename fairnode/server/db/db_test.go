package db

import (
	"fmt"
	"gopkg.in/mgo.v2/bson"
	"log"
	"testing"
	"time"
)

var session, _ = New("localhost", "27017", "", "")

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
