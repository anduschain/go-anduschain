package db

import (
	"fmt"
	"testing"
)

var session = New("", "", "")

func TestFairNodeDB_GetActiveNodeList(t *testing.T) {
	var result []activeNode
	session.ActiveNodeCol.Find(nil).All(&result)
	fmt.Println(result)
	defer session.Mongo.Close()
}
