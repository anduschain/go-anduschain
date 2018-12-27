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
	GetP2PServer() *p2p.Server
	GetCoinbase() common.Address
	SetOtprnWithSig(otprn *otprn.Otprn, sig []byte)
	GetOtprnWithSig() *types.OtprnWithSig
	GetCurrentBalance() *big.Int
	GetCurrentJoinNonce() uint64
	GetTxpool() *core.TxPool
	GetBlockChain() *core.BlockChain
	GetCoinbsePrivKey() *ecdsa.PrivateKey
	BlockMakeStart() chan struct{}
	VoteBlock() chan *fairtypes.VoteBlock
	FinalBlock() chan fairtypes.FinalBlock
	GetSigner() gethType.Signer
}
