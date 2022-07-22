package params

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/crypto"
	"math/big"
)

const (
	jtsAddrStr    = "0x000000000000000000000000000000000000da07"
	MinerGasCeil  = 8e10
	MinerGasFloor = 8e18
	DefaultGasFee = 5240000000000
)

var (
	JtxAddress         = common.HexToAddress(jtsAddrStr)
	TestOtprn          = "f886940eb16fc8e911b30cab0186546bc417376349f8ef9471562b71999873db5b286df957af199ec94617f780d78080823330808080cd30821518887fffffffffffffff80b8413a1dde25d1e9e6923a70877e71a9205170b8aa664d0bd6812db5b9290c73090d095cc94422abd6626e7d17567b4e7a7bbc9e0257e3acb9425656d32f95a866d601"
	TestFairnodeKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	TestFairnodeAddr   = crypto.PubkeyToAddress(TestFairnodeKey.PublicKey)
	TestDebConfig      = &DebConfig{
		FairPubKey: common.Bytes2Hex(crypto.CompressPubkey(&TestFairnodeKey.PublicKey)),
		GasLimit:   MaxGasLimit,
		GasPrice:   DefaultGasFee,
		FnFeeRate:  big.NewInt(100),
	}
	TestCliqueConfig = &CliqueConfig{
		Period: 1,
		Epoch:  50,
	}
	MainNetId = big.NewInt(14288640)
	TestNetId = big.NewInt(14288641)
	DvlpNetId = big.NewInt(14288642)
	GeneralId = big.NewInt(3357)
	SideId    = big.NewInt(3358)
)
