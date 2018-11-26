package db

import (
	"fmt"
	"github.com/mongodb/mongo-go-driver/bson"
	"testing"
)

var session = New("", "", "")

type enodeid2 struct {
	Enodeid string
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
