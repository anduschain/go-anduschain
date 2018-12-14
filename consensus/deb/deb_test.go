package deb

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"testing"
)

func TestMakeRand(t *testing.T) {

	//rand := big.NewInt(0)
	//
	//for i := 0; i <= int(10); i++ {
	//	newRand := big.NewInt(int64(i))
	//	if newRand.Cmp(rand) > 0 {
	//		rand = newRand
	//	}
	//
	//	fmt.Println(rand.String(), newRand.String())
	//}

	otp := otprn.New(10)
	otprnHash := otp.HashOtprn()
	pHash := common.HexToHash("0xcc1df0353d0f711866a6782a97599656cb3c95a4c903dfde626553b1b864b959")
	miner := common.HexToAddress("0x0acfeda530894ac947ed5c8f30b9afdac347c3e7")

	result := MakeRand(uint64(0), otprnHash, miner, pHash)

	fmt.Println(result.String(), result.Uint64())

}
