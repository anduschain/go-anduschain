package _interface

import (
	"crypto/ecdsa"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core"
	gethType "github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/client/types"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/p2p"
	"math/big"
)

type ServiceFunc interface {
	Start() error
	Stop() error
}

type Client interface {
	SetBlockMine(status bool)
	GetBlockMine() bool
	GetP2PServer() *p2p.Server
	GetCoinbase() common.Address
	SetOtprnWithSig(otprn *otprn.Otprn, sig []byte)
	GetOtprnWithSig(otprnHash common.Hash) *types.OtprnWithSig
	SetCurrnetOtprnHash(otprnHash common.Hash)
	GetCurrnetOtprnHash() common.Hash
	GetSavedOtprnHashs() []common.Hash
	GetCurrentBalance() *big.Int
	GetCurrentJoinNonce() uint64
	GetTxpool() *core.TxPool
	GetBlockChain() *core.BlockChain
	GetCoinbsePrivKey() *ecdsa.PrivateKey
	BlockMakeStart() chan struct{}
	VoteBlock() chan *fairtypes.Vote
	FinalBlock() chan fairtypes.FinalBlock
	GetSigner() gethType.Signer
	GetCurrentNonce(addr common.Address) uint64
	SaveWiningBlock(block *gethType.Block)
	GetWinningBlock(hash common.Hash) *gethType.Block
}
