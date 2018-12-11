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

	voters := []Coinbase{
		Coinbase(common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")), // 1
		Coinbase(common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87")), // 1
		Coinbase(common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d88")), // 2
		Coinbase(common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d89")), // 3
		Coinbase(common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d90")), // 4
		Coinbase(common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d90")), // 4
		Coinbase(common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d91")), // 5
		Coinbase(common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d92")), // 6
		Coinbase(common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d93")), // 7
	}

	for i, block := range blocks {
		//headers[i] = block.Header()
		pool.InsertCh <- Vote{
			hash,
			*block,
			voters[i],
		}

	}

	for i := range voters {
		pool.InsertCh <- Vote{
			hash,
			*blocks[0],
			voters[i],
		}
	}

	for i := range pool.pool[hash] {
		fmt.Println(pool.pool[hash][i].Count)
	}

	pool.Stop()

}
