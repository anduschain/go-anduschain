package pool

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/consensus/ethash"
	"github.com/anduschain/go-anduschain/core"
	"github.com/anduschain/go-anduschain/ethdb"
	"github.com/anduschain/go-anduschain/params"
	"testing"
)

func TestNewVotePool(t *testing.T) {

	var (
		testdb    = ethdb.NewMemDatabase()
		gspec     = &core.Genesis{Config: params.TestChainConfig}
		genesis   = gspec.MustCommit(testdb)
		blocks, _ = core.GenerateChain(params.TestChainConfig, genesis, ethash.NewFaker(), testdb, 8, nil)
	)

	//headers := make([]*types.Header, len(blocks))

	pool := NewVotePool(nil)

	pool.Start()

	hash := OtprnHash("0x123asd456")
	hash2 := OtprnHash("0x123asd458")

	//voters := []common.Address{
	//	common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"), // 1
	//	common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"), // 1
	//	common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d88"), // 2
	//	common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d89"), // 3
	//	common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d90"), // 4
	//	common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d90"), // 4
	//	common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d91"), // 5
	//	common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d92"), // 6
	//	common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d93"), // 7
	//}

	test := []struct {
		hash OtprnHash
		addr common.Address
	}{
		{OtprnHash("0x123asd456"), common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")}, // 1
		{OtprnHash("0x123asd456"), common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")}, // 1

		{OtprnHash("0x123asd458"), common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")}, //   1
		{OtprnHash("0x123asd458"), common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")}, //   1

		{OtprnHash("0x123asd456"), common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d88")}, // 2
		{OtprnHash("0x123asd456"), common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d89")}, // 3

		{OtprnHash("0x123asd458"), common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d88")}, //   2
		{OtprnHash("0x123asd458"), common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d89")}, //   3
	}

	fmt.Println("---Block count-------", len(blocks))

	for i, block := range blocks {
		//headers[i] = block.Header()
		pool.InsertCh <- Vote{
			test[i].hash,
			block,
			test[i].addr,
		}

	}

	//for i := range voters {
	//	pool.InsertCh <- Vote{
	//		hash,
	//		blocks[0],
	//		voters[i],
	//	}
	//}

	//fmt.Println("----hash---")
	//
	//for i := range pool.pool[hash] {
	//	fmt.Println(pool.pool[hash][i].Count)
	//}
	//
	//fmt.Println("----hash2---")
	//
	//for i := range pool.pool[hash2] {
	//	fmt.Println(pool.pool[hash][i].Count)
	//}

	vblocks := pool.GetVoteBlocks(hash)
	fmt.Println(len(vblocks))

	vblocks2 := pool.GetVoteBlocks(hash2)
	fmt.Println(len(vblocks2))

	pool.Stop()

}
