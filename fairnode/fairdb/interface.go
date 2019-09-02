package fairdb

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/pkg/errors"
	log "gopkg.in/inconshreveable/log15.v2"
	"math/big"
)

var logger = log.New("fairnode", "database")

var (
	ErrAlreadyExistBlock = errors.New("already exist block")
)

type FairnodeDB interface {
	Start() error
	Stop()

	GetChainConfig() *types.ChainConfig
	SaveChainConfig(config *types.ChainConfig) error

	CurrentInfo() *types.CurrentInfo
	CurrentBlock() *types.Block
	CurrentOtprn() *types.Otprn

	InitActiveNode()
	SaveActiveNode(node types.HeartBeat)
	GetActiveNode() []types.HeartBeat
	RemoveActiveNode(enode string)

	SaveOtprn(otprn types.Otprn)
	GetOtprn(otprnHash common.Hash) *types.Otprn

	SaveLeague(otprnHash common.Hash, enode string)
	GetLeagueList(otprnHash common.Hash) []types.HeartBeat

	SaveVote(otprn common.Hash, blockNum *big.Int, vote *types.Voter)
	GetVoters(votekey common.Hash) []*types.Voter

	SaveFinalBlock(block *types.Block, byteBlock []byte) error
	GetBlock(blockHash common.Hash) *types.Block
	RemoveBlock(blockHash common.Hash)
}

func MakeVoteKey(otprn common.Hash, blockNum *big.Int) common.Hash {
	var k []byte
	k = append(k, otprn.Bytes()...)
	k = append(k, blockNum.Bytes()...)
	return common.BytesToHash(k)
}
