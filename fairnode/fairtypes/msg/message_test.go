package msg

import (
	"fmt"
	"testing"
)

func TestSend(t *testing.T) {
	type testStruct struct {
		Name string
		Age  uint64
	}

	result, err := makeMassage(SendOTPRN, &testStruct{"hakuna", 19})
	if err != nil {
		t.Error(err)
	}

	var ts testStruct
	m := ReadMsg(result)
	m.Decode(&ts)

	fmt.Println(m, ts)
}
