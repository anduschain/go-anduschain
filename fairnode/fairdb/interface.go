package fairdb

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	log "gopkg.in/inconshreveable/log15.v2"
	"math/big"
)

var logger = log.New("fairnode", "database")

type FairnodeDB interface {
	Start() error
	Stop()

	GetChainConfig() *types.ChainConfig
	SaveChainConfig(config *types.ChainConfig) error

	CurrentBlock() *types.Block
	CurrentOtprn() *types.Otprn

	InitActiveNode()
	SaveActiveNode(node types.HeartBeat)
	GetActiveNode() []types.HeartBeat
	RemoveActiveNode(enode string)

	SaveOtprn(otprn types.Otprn)
	GetOtprn(otprnHash common.Hash) *types.Otprn

	SaveLeagueList(otprnHash common.Hash, lists []types.HeartBeat)
	GetLeagueList(otprnHash common.Hash) []types.HeartBeat

	SaveVote(otprn common.Hash, blockNum *big.Int, vote types.Voter)
	GetVoters(otekey common.Hash) []types.Voter

	SaveFinalBlock(block types.Block)
	GetBlock(blockHash common.Hash) *types.Block
}

func makeVoteKey(otprn common.Hash, blockNum *big.Int) common.Hash {
	var k []byte
	k = append(k, otprn.Bytes()...)
	k = append(k, blockNum.Bytes()...)
	return common.BytesToHash(k)
}
