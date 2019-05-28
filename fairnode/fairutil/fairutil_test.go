package fairutil

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"testing"
)

func TestIsJoinOK(t *testing.T) {
	t.Log("OTPRN 생성")

	tOtprn := types.New(1, 100, 100, 100)
	//for i:=0; i < 4; i++ {
	//	tOtprn = append(tOtprn, *otprn.New(11))
	//}

	t.Log("Address 생성")
	tAddress := []common.Address{
		common.HexToAddress("0xd565fa535b187291bd7f87e4e4dc574058900dc6"),
		common.HexToAddress("0x3abaaf99c3dd851bc26dd9edd4b9972b07ce9728"),
		common.HexToAddress("0xd565fa535b187291bd7f87e4e4dc574058900dc6"),
		common.HexToAddress("0x3abaaf99c3dd851bc26dd9edd4b9972b07ce9728"),
	}

	t.Log("참여 여부 대상확인하는 테스트")
	for i := range tAddress {
		result := IsJoinOK(tOtprn, tAddress[i])
		fmt.Println(result)
	}

}

func TestByteTrimSize(t *testing.T) {
	//datas := [][]byte{
	//	common.HexToAddress("0xd565fa535b187291bd7f87e4e4dc574058900dc6").Bytes(),
	//	common.HexToAddress("0x3abaaf99c3dd851bc26dd9edd4b9972b07ce9728").Bytes(),
	//	common.HexToAddress("0xd565fa535b187291bd7f87e4e4dc574058900dc6").Bytes(),
	//	common.HexToAddress("0x3abaaf99c3dd851bc26dd9edd4b9972b07ce9728").Bytes(),
	//}

	var res []byte
	data := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	t.Log("data byte array length", data, len(data))

	result := ByteTrimSize(data, 3)
	for i := 0; i < len(result); i++ {
		kk := result[i]
		res = append(res, kk...)
	}

	t.Log(data, res)

	res = []byte{}
	data2 := common.HexToAddress("0xd565fa535b187291bd7f87e4e4dc574058900dc6")
	result2 := ByteTrimSize(data2.Bytes(), 3)

	for i := 0; i < len(result2); i++ {
		res = append(res, result2[i]...)
	}

	t.Log(data2, res)

	if common.BytesToAddress(res) == data2 {
		t.Log("어드레스가 일치함")
	} else {
		t.Log("어드레스 불일치")
	}
}
