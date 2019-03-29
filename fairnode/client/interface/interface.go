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
	"github.com/anduschain/go-anduschain/p2p/nat"
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

	SaveWiningBlock(otprnHash common.Hash, block *gethType.Block)
	GetWinningBlock(otprnHash common.Hash, hash common.Hash) *gethType.Block
	DelWinningBlock(otprnHash common.Hash)

	StoreOtprnWidthSig(otprn *otprn.Otprn, sig []byte)
	GetStoreOtprnWidthSig() *otprn.Otprn
	GetUsingOtprnWithSig() *types.OtprnWithSig
	GetSavedOtprnHashs() []common.Hash
	FindOtprn(otprnHash common.Hash) *types.OtprnWithSig

	GetNat() nat.Interface

	WinningBlockVoteStart() chan struct{}
}
