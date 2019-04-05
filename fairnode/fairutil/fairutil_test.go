package fairutil

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"testing"
)

func TestIsJoinOK(t *testing.T) {
	t.Log("OTPRN 생성")

	tOtprn := otprn.New(1, 100, 100, 100)
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
		result := IsJoinOK(*tOtprn, tAddress[i])
		fmt.Println(result)
	}

}
