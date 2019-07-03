package _interface

import (
	"crypto/ecdsa"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
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
	GetSigner() types.Signer
	GetCurrentNonce(addr common.Address) uint64

	SaveWiningBlock(otprnHash common.Hash, block *types.Block)
	GetWinningBlock(otprnHash common.Hash, hash common.Hash) *types.Block
	DelWinningBlock(otprnHash common.Hash)

	StoreOtprnWidthSig(otprn *types.Otprn, sig []byte)
	DeleteStoreOtprnWidthSig()
	GetStoreOtprnWidthSig() *types.Otprn
	GetUsingOtprnWithSig() *types.Otprn
	GetSavedOtprnHashs() []common.Hash
	FindOtprn(otprnHash common.Hash) *types.Otprn

	GetNat() nat.Interface

	WinningBlockVoteStart() chan struct{}
}
