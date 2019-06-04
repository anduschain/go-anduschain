package pool

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/consensus/deb"
	"github.com/anduschain/go-anduschain/core"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/ethdb"
	"github.com/anduschain/go-anduschain/params"
	"testing"
	"time"
)

func TestNewVotePool(t *testing.T) {

	var (
		testdb       = ethdb.NewMemDatabase()
		gspec        = &core.Genesis{Config: params.TestChainConfig}
		genesis      = gspec.MustCommit(testdb)
		blocks, _, _ = core.GenerateChain(params.TestChainConfig, genesis, deb.NewFaker(), testdb, 8, nil)
	)

	//headers := make([]*types.Header, len(blocks))

	pool := NewVotePool(nil)

	pool.Start()

	hash := OtprnHash(common.BytesToHash([]byte("0x123asd456")))
	addr1 := "0x2177CC24C2eEf8aBD18Fd27214B7d9a79fA2B749"
	addr2 := "0x47dffCF319F986E658B61287644b1b6127D2b9c3"

	votes := []Vote{
		{hash, blocks[0].Hash(), types.Voter{common.HexToAddress(addr1), []byte("0x47dffCF319F986E658B61287644b1b6127D2b9c3"), "diffcult"}}, // 없음
		{hash, blocks[0].Hash(), types.Voter{common.HexToAddress(addr2), []byte("0x47dffCF319F986E658B61287644b1b6127D2b9c3"), "diffcult"}}, // 있음 ,중복 아님
		{hash, blocks[1].Hash(), types.Voter{common.HexToAddress(addr1), []byte("0x47dffCF319F986E658B61287644b1b6127D2b9c3"), "diffcult"}}, // 없음
		{hash, blocks[1].Hash(), types.Voter{common.HexToAddress(addr2), []byte("0x47dffCF319F986E658B61287644b1b6127D2b9c3"), "diffcult"}}, // 있음 ,중복 아님
	}

	for i := range votes {
		pool.InsertCh <- votes[i]
	}

	time.Sleep(2 * time.Second)

	if val, ok := pool.pool[hash]; ok {
		for i := range val {
			fmt.Println("-------", val[i].Count, val[i].Voters)
		}

	}

	pool.Stop()

}
