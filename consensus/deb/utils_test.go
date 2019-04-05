package deb

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"math/big"
	"math/rand"
	"testing"
)

type testAddr struct {
	Addr      common.Address
	JoinNonce uint64
	SucCount  uint64
}

type ResAttemp struct {
	Addr     string
	SucCount uint64 // Max diff 생성 횟수
}

type Result struct {
	Attempt int64
	AttList []ResAttemp
}

var (
	testCount        = 1000
	Mminer    uint64 = 10
	addresses        = []testAddr{
		{common.HexToAddress("0x095e7baea6a6c7c4c2dfeb977efac326af552d87"), 0, 0},
		{common.HexToAddress("0x2cac1adea150210703ba75ed097ddfe24e14f213"), 0, 0},
		{common.HexToAddress("0x8bda78331c916a08481428e4b07c96d3e916d165"), 0, 0},
		{common.HexToAddress("0xd49ff4eeb0b2686ed89c0fc0f2b6ea533ddbbd5e"), 0, 0},
		{common.HexToAddress("0x7ef5a6135f1fd6a02593eedc869c6d41d934aef8"), 0, 0},
		{common.HexToAddress("0xc8E4E5c809fF71c3742022b96DB8615F970A5A3b"), 0, 0},
		{common.HexToAddress("0x0C81208f82e698cFe3Cf8229f8BCB98a72394b9c"), 0, 0},
		{common.HexToAddress("0x7E257B3FeBa1406fC7271383d37BBc224D755563"), 0, 0},
		{common.HexToAddress("0x950365918a07C7254a71A162fae80e5533538a02"), 0, 0},
		{common.HexToAddress("0x73C62c514A351329Ec0E024e37fb3124bEA87b1f"), 0, 0},
	}
	otprnHash []common.Hash
	blockHash = []common.Hash{
		common.HexToHash("0x807e4fe9328a0ee47d89ee3607d8a23446a27f612b51cf21fc96ea5e8b5249be"),
		common.HexToHash("0xc487bcda49c5d8420e8672258c19dde1a5eb7a25bb40acb4b4f2b12f69e86444"),
		common.HexToHash("0xb217f086d88b2dca2c3e49b716f674220ffdb540970fd1aced611f68107e82d9"),
		common.HexToHash("0x8558f263a3357dde7fc6eb116996500ddda3d797fa211fb746c29bfbbf675286"),
		common.HexToHash("0x5f56377915b65f8f331d91eca94fb6f3aa2c495f27df5006ab55ba69f1edee4d"),
		common.HexToHash("0xe334f05a70bf2a8bc01809ab896c563ac203a941d425f7b080bfff41ba4cefea"),
		common.HexToHash("0xa2b050b00d4040ce3b8e8d525b981e8cb56088c5fb7f6ceea7de7f0816165426"),
		common.HexToHash("0x2c0c58bc7c271d567b7ab439eeb82ed1b10b52bd009444a26cb5d448cb571d04"),
		common.HexToHash("0xfbb2ff40f515480df4539bca6954d0366f2dfd705467966d220fbdc0a710cb9b"),
		common.HexToHash("0x2c6acaaaa25ea1ee50e2b336aec05394e6e8bdfff3b120a5695e9c52871040b1"),
	}
	Results []Result
)

func init() {
	// otprn 생성
	makeOtprn()
}

func makeOtprn() {
	for i := 0; i < len(blockHash); i++ {
		otp := otprn.New(Mminer, 100, 100, 100)
		otprnHash = append(otprnHash, otp.HashOtprn())
	}
}

// balockHash, otprn > 통제변인
// joinnonce, address에 따른 top difficulty 생성 횟수로 결과값 확인 ( 골고루 분포 되는지 확인)
func TestDeb_MakeRand(t *testing.T) {
	for i := 0; i < testCount; i++ {
		var diff int64
		var addr common.Address
		r := rand.Intn(10)
		b := blockHash[r].Big().Int64() * big.NewInt(int64(r)).Int64()
		for i := range addresses {
			d := MakeRand(addresses[i].JoinNonce, otprnHash[0], addresses[i].Addr, common.BigToHash(big.NewInt(b)))
			if diff < d {
				diff = d
				addr = addresses[i].Addr
			}

			addresses[i].JoinNonce++
		}

		for i := range addresses {
			if addresses[i].Addr == addr {
				addresses[i].SucCount++
				addresses[i].JoinNonce = 0
			}
		}
	}

	for i := range addresses {
		t.Log("Address : ", addresses[i].Addr.String(), "JoinNonce : ", addresses[i].JoinNonce, "SucCount : ", addresses[i].SucCount)
	}

}
