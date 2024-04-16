// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package params

import (
	"encoding/binary"
	"fmt"
	"github.com/anduschain/go-anduschain/crypto"
	"golang.org/x/crypto/sha3"
	"math/big"

	"github.com/anduschain/go-anduschain/common"
)

// Genesis hashes to enforce below configs on.
var (
	MainnetGenesisHash = common.HexToHash("0x36f0e7a4f08e35a5ce5a624792e3253c21cbe103b037e276fe376bfb0ebc3f81")
	TestnetGenesisHash = common.HexToHash("0x2e673ece7dd30d2b04ede22c800f816c0c9ef2fe43433337565fa4e4c8405d67")

	MainNetPubKey = "034911d106851a0d857cb6aeaf39a06ce3d7a13f73195c8a58874156bb28d2ad0e" // from fairnode-mainnet
	TestNetPubKey = "02c5ec32bf37887175010ff7f8d89723fa7fe584408f4e39c51d9f26f685d50b79" // from fairnode-testnet

	TestPubKey = "028fb2276965f6a47de9cb36407eb2c9e158e2d4d93e5661e19b7d7b61209e1f87" // file in Projcet fiarkey.json, [ only using for testing ]
)

// chainID rule = 0xdao700 -> to dec 14288640 // mainnet
// chainID rule = 0xdao701 -> to dec 14288641 // testnet
// chainID rule = 0xdao702 -> to dec 14288640 // ...

var (
	MAIN_NETWORK = big.NewInt(14288640)
	TEST_NETWORK = big.NewInt(14288641)
	DEB_NETWORK  = big.NewInt(14288642)
	DVLP_NETWORK = big.NewInt(3355)
)

var (
	// MainnetChainConfig is the chain parameters to run a node on the main network.

	MainnetChainConfig = &ChainConfig{
		ChainID:             MAIN_NETWORK,
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.Hash{},
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PohangBlock:         big.NewInt(2000000),
		UlsanBlock:          big.NewInt(3800000),
		Deb:                 &DebConfig{FairPubKey: MainNetPubKey, GasLimit: GenesisGasLimit, GasPrice: MinimumGenesisGasPrice, FnFeeRate: big.NewInt(DefaultFairnodeFee)},
	}

	// TestnetChainConfig contains the chain parameters to run a node on the Anduschain test network.
	TestnetChainConfig = &ChainConfig{
		ChainID:             TEST_NETWORK,
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.Hash{},
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PohangBlock:         big.NewInt(0),
		UlsanBlock:          big.NewInt(0),
		Deb:                 &DebConfig{FairPubKey: TestNetPubKey, GasLimit: GenesisGasLimit, GasPrice: MinimumGenesisGasPrice, FnFeeRate: big.NewInt(DefaultFairnodeFee)},
	}

	DebChainConfig = &ChainConfig{
		ChainID:             DEB_NETWORK,
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      true,
		EIP150Block:         nil,
		EIP150Hash:          common.Hash{},
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PohangBlock:         big.NewInt(0),
		UlsanBlock:          big.NewInt(0),
		Deb: &DebConfig{
			FairPubKey: TestPubKey,
			GasLimit:   GenesisGasLimit,
			GasPrice:   MinimumGenesisGasPrice,
			FnFeeRate:  big.NewInt(DefaultFairnodeFee),
		},
	}

	TestChainConfig = &ChainConfig{
		GeneralId, big.NewInt(0), nil, true,
		big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0),
		big.NewInt(0), TestDebConfig, nil, nil,
		ScrollConfig{
			UseZktrie:                 false,
			FeeVaultAddress:           &common.Address{123},
			EnableEIP2718:             true,
			EnableEIP1559:             true,
			MaxTxPerBlock:             nil,
			MaxTxPayloadBytesPerBlock: nil,
			L1Config:                  &L1Config{5, common.HexToAddress("0x0000000000000000000000000000000000000000"), 0, common.HexToAddress("0x0000000000000000000000000000000000000000")},
		}}

	TestDebChainConfig = &ChainConfig{
		DvlpNetId, big.NewInt(0), nil, true,
		big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0),
		big.NewInt(0), TestDebConfig, nil, nil,
		ScrollConfig{
			UseZktrie:                 false,
			FeeVaultAddress:           &common.Address{123},
			EnableEIP2718:             true,
			EnableEIP1559:             true,
			MaxTxPerBlock:             nil,
			MaxTxPayloadBytesPerBlock: nil,
			L1Config:                  &L1Config{5, common.HexToAddress("0x0000000000000000000000000000000000000000"), 0, common.HexToAddress("0x0000000000000000000000000000000000000000")},
		}}

	AllDebProtocolChanges = &ChainConfig{
		DvlpNetId, big.NewInt(0), nil, true,
		big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0),
		big.NewInt(0), TestDebConfig, nil, nil,
		ScrollConfig{
			UseZktrie:                 false,
			FeeVaultAddress:           &common.Address{123},
			EnableEIP2718:             true,
			EnableEIP1559:             true,
			MaxTxPerBlock:             nil,
			MaxTxPayloadBytesPerBlock: nil,
			L1Config:                  &L1Config{5, common.HexToAddress("0x0000000000000000000000000000000000000000"), 0, common.HexToAddress("0x0000000000000000000000000000000000000000")},
		}}

	TestCliqueChainConfig = &ChainConfig{
		GeneralId, big.NewInt(0), nil, true,
		big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0),
		big.NewInt(0), big.NewInt(0), big.NewInt(0),
		big.NewInt(0), nil, &CliqueConfig{Period: 0, Epoch: 30000}, nil,
		ScrollConfig{
			UseZktrie:                 false,
			FeeVaultAddress:           &common.Address{123},
			EnableEIP2718:             true,
			EnableEIP1559:             true,
			MaxTxPerBlock:             nil,
			MaxTxPayloadBytesPerBlock: nil,
			L1Config:                  &L1Config{5, common.HexToAddress("0x0000000000000000000000000000000000000000"), 0, common.HexToAddress("0x0000000000000000000000000000000000000000")},
		}}

	AllCliqueProtocolChanges = &ChainConfig{
		big.NewInt(1337), big.NewInt(0), nil, false,
		big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0),
		big.NewInt(0), big.NewInt(0), big.NewInt(0),
		big.NewInt(0), nil, &CliqueConfig{Period: 0, Epoch: 30000}, nil,
		ScrollConfig{
			UseZktrie:                 false,
			FeeVaultAddress:           &common.Address{123},
			EnableEIP2718:             true,
			EnableEIP1559:             true,
			MaxTxPerBlock:             nil,
			MaxTxPayloadBytesPerBlock: nil,
			L1Config:                  &L1Config{5, common.HexToAddress("0x0000000000000000000000000000000000000000"), 0, common.HexToAddress("0x0000000000000000000000000000000000000000")},
		}}

	TestSseChainConfig = &ChainConfig{
		GeneralId, big.NewInt(0), nil, true,
		big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0),
		big.NewInt(0), big.NewInt(0), big.NewInt(0),
		big.NewInt(0), nil, nil, &SseConfig{Period: 0, Epoch: 30000},
		ScrollConfig{
			UseZktrie:                 false,
			FeeVaultAddress:           &common.Address{123},
			EnableEIP2718:             true,
			EnableEIP1559:             true,
			MaxTxPerBlock:             nil,
			MaxTxPayloadBytesPerBlock: nil,
			L1Config:                  &L1Config{5, common.HexToAddress("0x0000000000000000000000000000000000000000"), 0, common.HexToAddress("0x0000000000000000000000000000000000000000")},
		}}

	AllSseProtocolChanges = &ChainConfig{
		big.NewInt(1337), big.NewInt(0), nil, false,
		big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0),
		big.NewInt(0), big.NewInt(0), big.NewInt(0),
		big.NewInt(0), nil, nil, &SseConfig{Period: 0, Epoch: 30000},
		ScrollConfig{
			UseZktrie:                 false,
			FeeVaultAddress:           &common.Address{123},
			EnableEIP2718:             true,
			EnableEIP1559:             true,
			MaxTxPerBlock:             nil,
			MaxTxPayloadBytesPerBlock: nil,
			L1Config:                  &L1Config{5, common.HexToAddress("0x0000000000000000000000000000000000000000"), 0, common.HexToAddress("0x0000000000000000000000000000000000000000")},
		}}
)

// ChainConfig is the core config which determines the blockchain settings.
//
// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig struct {
	ChainID *big.Int `json:"chainId"` // chainId identifies the current chain and is used for replay protection

	HomesteadBlock *big.Int `json:"homesteadBlock,omitempty"` // Homestead switch block (nil = no fork, 0 = already homestead)

	DAOForkBlock   *big.Int `json:"daoForkBlock,omitempty"`   // TheDAO hard-fork switch block (nil = no fork)
	DAOForkSupport bool     `json:"daoForkSupport,omitempty"` // Whether the nodes supports or opposes the DAO hard-fork

	// EIP150 implements the Gas price changes (https://github.com/ethereum/EIPs/issues/150)
	EIP150Block *big.Int    `json:"eip150Block,omitempty"` // EIP150 HF block (nil = no fork)
	EIP150Hash  common.Hash `json:"eip150Hash,omitempty"`  // EIP150 HF hash (needed for header only clients as only gas pricing changed)

	EIP155Block *big.Int `json:"eip155Block,omitempty"` // EIP155 HF block
	EIP158Block *big.Int `json:"eip158Block,omitempty"` // EIP158 HF block

	ByzantiumBlock      *big.Int `json:"byzantiumBlock,omitempty"`      // Byzantium switch block (nil = no fork, 0 = already on byzantium)
	ConstantinopleBlock *big.Int `json:"constantinopleBlock,omitempty"` // Constantinople switch block (nil = no fork, 0 = already activated)
	PohangBlock         *big.Int `json:"pohangBlock,omitempty"`         // Pohang switch block (nil = no fork, 0 = already activated)
	UlsanBlock          *big.Int `json:"ulsanBlock,omitempty"`          // Pohang switch block (nil = no fork, 0 = already activated)

	// Various consensus engines
	Deb    *DebConfig    `json:"deb,omitempty"`
	Clique *CliqueConfig `json:"clique,omitempty"`
	Sse    *SseConfig    `json:"sse,omitempty"`

	// Scroll genesis extension: enable scroll rollup-related traces & state transition
	Scroll ScrollConfig `json:"scroll,omitempty"`
}

type ScrollConfig struct {
	// Use zktrie [optional]
	UseZktrie bool `json:"useZktrie,omitempty"`

	// Maximum number of transactions per block [optional]
	MaxTxPerBlock *int `json:"maxTxPerBlock,omitempty"`

	// Maximum tx payload size of blocks that we produce [optional]
	MaxTxPayloadBytesPerBlock *int `json:"maxTxPayloadBytesPerBlock,omitempty"`

	// Transaction fee vault address [optional]
	FeeVaultAddress *common.Address `json:"feeVaultAddress,omitempty"`

	// Enable EIP-2718 in tx pool [optional]
	EnableEIP2718 bool `json:"enableEIP2718,omitempty"`

	// Enable EIP-1559 in tx pool, EnableEIP2718 should be true too [optional]
	EnableEIP1559 bool `json:"enableEIP1559,omitempty"`

	// L1 config
	L1Config *L1Config `json:"l1Config,omitempty"`
}

// L1Config contains the l1 parameters needed to sync l1 contract events (e.g., l1 messages, commit/revert/finalize batches) in the sequencer
type L1Config struct {
	L1ChainId             uint64         `json:"l1ChainId,string,omitempty"`
	L1MessageQueueAddress common.Address `json:"l1MessageQueueAddress,omitempty"`
	NumL1MessagesPerBlock uint64         `json:"numL1MessagesPerBlock,string,omitempty"`
	ScrollChainAddress    common.Address `json:"scrollChainAddress,omitempty"`
}

func (c *L1Config) String() string {
	if c == nil {
		return "<nil>"
	}

	return fmt.Sprintf("{l1ChainId: %v, l1MessageQueueAddress: %v, numL1MessagesPerBlock: %v, ScrollChainAddress: %v}",
		c.L1ChainId, c.L1MessageQueueAddress.Hex(), c.NumL1MessagesPerBlock, c.ScrollChainAddress.Hex())
}

func (s ScrollConfig) BaseFeeEnabled() bool {
	return s.EnableEIP2718 && s.EnableEIP1559
}

func (s ScrollConfig) FeeVaultEnabled() bool {
	return s.FeeVaultAddress != nil
}

func (s ScrollConfig) ZktrieEnabled() bool {
	return s.UseZktrie
}

func (s ScrollConfig) ShouldIncludeL1Messages() bool {
	return s.L1Config != nil && s.L1Config.NumL1MessagesPerBlock > 0
}

func (s ScrollConfig) String() string {
	maxTxPerBlock := "<nil>"
	if s.MaxTxPerBlock != nil {
		maxTxPerBlock = fmt.Sprintf("%v", *s.MaxTxPerBlock)
	}

	maxTxPayloadBytesPerBlock := "<nil>"
	if s.MaxTxPayloadBytesPerBlock != nil {
		maxTxPayloadBytesPerBlock = fmt.Sprintf("%v", *s.MaxTxPayloadBytesPerBlock)
	}

	return fmt.Sprintf("{useZktrie: %v, maxTxPerBlock: %v, MaxTxPayloadBytesPerBlock: %v, feeVaultAddress: %v, enableEIP2718: %v, enableEIP1559: %v, l1Config: %v}",
		s.UseZktrie, maxTxPerBlock, maxTxPayloadBytesPerBlock, s.FeeVaultAddress, s.EnableEIP2718, s.EnableEIP1559, s.L1Config.String())
}

// IsValidTxCount returns whether the given block's transaction count is below the limit.
// This limit corresponds to the number of ECDSA signature checks that we can fit into the zkEVM.
func (s ScrollConfig) IsValidTxCount(count int) bool {
	return s.MaxTxPerBlock == nil || count <= *s.MaxTxPerBlock
}

// IsValidBlockSize returns whether the given block's transaction payload size is below the limit.
func (s ScrollConfig) IsValidBlockSize(size common.StorageSize) bool {
	return s.MaxTxPayloadBytesPerBlock == nil || size <= common.StorageSize(*s.MaxTxPayloadBytesPerBlock)
}

type DebConfig struct {
	FairPubKey string   `json:"fairPubKey"`
	GasLimit   uint64   `json:"gasLimit"`
	GasPrice   uint64   `json:"gasPrice"`
	FnFeeRate  *big.Int `json:"fairFee"`
}

func (c *DebConfig) FairAddr() common.Address {
	// GenesisBlock에서 FairPubKey는 세팅됨
	fnPubKey, err := crypto.DecompressPubkey(common.Hex2Bytes(c.FairPubKey))
	if err != nil {
		return common.Address{}
	}
	return crypto.PubkeyToAddress(*fnPubKey)
}

func (c *DebConfig) SetFnFeeRate(fairFeeRate *big.Int) {
	c.FnFeeRate = fairFeeRate
}

func (c *DebConfig) SetFairPubKey(fairPubKey string) {
	c.FairPubKey = fairPubKey
}

func (c *DebConfig) GetFnFeeRate() *big.Int {
	return c.FnFeeRate
}

func (c *DebConfig) String() string {
	return "deb"
}

// CliqueConfig is the consensus engine configs for proof-of-authority based sealing.
type CliqueConfig struct {
	Period uint64 `json:"period"` // Number of seconds between blocks to enforce
	Epoch  uint64 `json:"epoch"`  // Epoch length to reset votes and checkpoint
}

// String implements the stringer interface, returning the consensus engine details.
func (c *CliqueConfig) String() string {
	return "clique"
}

// SseConfig is the consensus engine configs for proof-of-authority based sealing.
type SseConfig struct {
	Period uint64 `json:"period"` // Number of seconds between blocks to enforce
	Epoch  uint64 `json:"epoch"`  // Epoch length to reset votes and checkpoint
}

// String implements the stringer interface, returning the consensus engine details.
func (c *SseConfig) String() string {
	return "sse"
}

// String implements the fmt.Stringer interface.
func (c *ChainConfig) String() string {
	var engine interface{}
	switch {
	case c.Deb != nil:
		engine = c.Deb
	case c.Clique != nil:
		engine = c.Clique
	case c.Sse != nil:
		engine = c.Sse
	default:
		engine = "unknown"
	}
	return fmt.Sprintf("{ChainID: %v Homestead: %v DAO: %v DAOSupport: %v EIP150: %v EIP155: %v EIP158: %v Byzantium: %v Constantinople: %v Engine: %v}",
		c.ChainID,
		c.HomesteadBlock,
		c.DAOForkBlock,
		c.DAOForkSupport,
		c.EIP150Block,
		c.EIP155Block,
		c.EIP158Block,
		c.ByzantiumBlock,
		c.ConstantinopleBlock,
		engine,
	)
}

// return newtork type
func (c *ChainConfig) NetworkType() uint64 {
	if c.ChainID.Cmp(MAIN_NETWORK) == 0 {
		return 0
	}
	if c.ChainID.Cmp(TEST_NETWORK) == 0 {
		return 1
	}
	return 2
}

// IsHomestead returns whether num is either equal to the homestead block or greater.
func (c *ChainConfig) IsHomestead(num *big.Int) bool {
	return isForked(c.HomesteadBlock, num)
}

// IsDAOFork returns whether num is either equal to the DAO fork block or greater.
func (c *ChainConfig) IsDAOFork(num *big.Int) bool {
	return isForked(c.DAOForkBlock, num)
}

// IsEIP150 returns whether num is either equal to the EIP150 fork block or greater.
func (c *ChainConfig) IsEIP150(num *big.Int) bool {
	return isForked(c.EIP150Block, num)
}

// IsEIP155 returns whether num is either equal to the EIP155 fork block or greater.
func (c *ChainConfig) IsEIP155(num *big.Int) bool {
	return isForked(c.EIP155Block, num)
}

// IsEIP158 returns whether num is either equal to the EIP158 fork block or greater.
func (c *ChainConfig) IsEIP158(num *big.Int) bool {
	return isForked(c.EIP158Block, num)
}

// IsByzantium returns whether num is either equal to the Byzantium fork block or greater.
func (c *ChainConfig) IsByzantium(num *big.Int) bool {
	return isForked(c.ByzantiumBlock, num)
}

// IsConstantinople returns whether num is either equal to the Constantinople fork block or greater.
func (c *ChainConfig) IsConstantinople(num *big.Int) bool {
	return isForked(c.ConstantinopleBlock, num)
}

// IsPohang returns whether num is either equal to the Pohang fork block or greater.
func (c *ChainConfig) IsPohang(num *big.Int) bool {
	if c.ChainID.Cmp(DVLP_NETWORK) == 0 {
		return true
	} else {
		return isForked(c.PohangBlock, num)
	}
}

// IsUlsan returns whether num is either equal to the Pohang fork block or greater.
func (c *ChainConfig) IsUlsan(num *big.Int) bool {
	if c.ChainID.Cmp(DVLP_NETWORK) == 0 {
		if num.Cmp(big.NewInt(160000)) > 0 {
			return true
		} else {
			return false
		}
	} else {
		return isForked(c.UlsanBlock, num)
	}

}

// GasTable returns the gas table corresponding to the current phase (homestead or homestead reprice).
//
// The returned GasTable's fields shouldn't, under any circumstances, be changed.
func (c *ChainConfig) GasTable(num *big.Int) GasTable {
	if num == nil {
		return GasTableHomestead
	}
	switch {
	case c.IsConstantinople(num):
		return GasTableConstantinople
	case c.IsEIP158(num):
		return GasTableEIP158
	case c.IsEIP150(num):
		return GasTableEIP150
	default:
		return GasTableHomestead
	}
}

// CheckCompatible checks whether scheduled fork transitions have been imported
// with a mismatching chain configuration.
func (c *ChainConfig) CheckCompatible(newcfg *ChainConfig, height uint64) *ConfigCompatError {
	bhead := new(big.Int).SetUint64(height)

	// Iterate checkCompatible to find the lowest conflict.
	var lasterr *ConfigCompatError
	for {
		err := c.checkCompatible(newcfg, bhead)
		if err == nil || (lasterr != nil && err.RewindTo == lasterr.RewindTo) {
			break
		}
		lasterr = err
		bhead.SetUint64(err.RewindTo)
	}
	return lasterr
}

func (c *ChainConfig) checkCompatible(newcfg *ChainConfig, head *big.Int) *ConfigCompatError {
	if isForkIncompatible(c.HomesteadBlock, newcfg.HomesteadBlock, head) {
		return newCompatError("Homestead fork block", c.HomesteadBlock, newcfg.HomesteadBlock)
	}
	if isForkIncompatible(c.DAOForkBlock, newcfg.DAOForkBlock, head) {
		return newCompatError("DAO fork block", c.DAOForkBlock, newcfg.DAOForkBlock)
	}
	if c.IsDAOFork(head) && c.DAOForkSupport != newcfg.DAOForkSupport {
		return newCompatError("DAO fork support flag", c.DAOForkBlock, newcfg.DAOForkBlock)
	}
	if isForkIncompatible(c.EIP150Block, newcfg.EIP150Block, head) {
		return newCompatError("EIP150 fork block", c.EIP150Block, newcfg.EIP150Block)
	}
	if isForkIncompatible(c.EIP155Block, newcfg.EIP155Block, head) {
		return newCompatError("EIP155 fork block", c.EIP155Block, newcfg.EIP155Block)
	}
	if isForkIncompatible(c.EIP158Block, newcfg.EIP158Block, head) {
		return newCompatError("EIP158 fork block", c.EIP158Block, newcfg.EIP158Block)
	}
	if c.IsEIP158(head) && !configNumEqual(c.ChainID, newcfg.ChainID) {
		return newCompatError("EIP158 chain ID", c.EIP158Block, newcfg.EIP158Block)
	}
	if isForkIncompatible(c.ByzantiumBlock, newcfg.ByzantiumBlock, head) {
		return newCompatError("Byzantium fork block", c.ByzantiumBlock, newcfg.ByzantiumBlock)
	}
	if isForkIncompatible(c.ConstantinopleBlock, newcfg.ConstantinopleBlock, head) {
		return newCompatError("Constantinople fork block", c.ConstantinopleBlock, newcfg.ConstantinopleBlock)
	}
	return nil
}

// isForkIncompatible returns true if a fork scheduled at s1 cannot be rescheduled to
// block s2 because head is already past the fork.
func isForkIncompatible(s1, s2, head *big.Int) bool {
	return (isForked(s1, head) || isForked(s2, head)) && !configNumEqual(s1, s2)
}

// isForked returns whether a fork scheduled at block s is active at the given head block.
func isForked(s, head *big.Int) bool {
	if s == nil || head == nil {
		return false
	}
	return s.Cmp(head) <= 0
}

func configNumEqual(x, y *big.Int) bool {
	if x == nil {
		return y == nil
	}
	if y == nil {
		return x == nil
	}
	return x.Cmp(y) == 0
}

// ConfigCompatError is raised if the locally-stored blockchain is initialised with a
// ChainConfig that would alter the past.
type ConfigCompatError struct {
	What string
	// block numbers of the stored and new configurations
	StoredConfig, NewConfig *big.Int
	// the block number to which the local chain must be rewound to correct the error
	RewindTo uint64
}

func newCompatError(what string, storedblock, newblock *big.Int) *ConfigCompatError {
	var rew *big.Int
	switch {
	case storedblock == nil:
		rew = newblock
	case newblock == nil || storedblock.Cmp(newblock) < 0:
		rew = storedblock
	default:
		rew = newblock
	}
	err := &ConfigCompatError{what, storedblock, newblock, 0}
	if rew != nil && rew.Sign() > 0 {
		err.RewindTo = rew.Uint64() - 1
	}
	return err
}

func (err *ConfigCompatError) Error() string {
	return fmt.Sprintf("mismatching %s in database (have %d, want %d, rewindto %d)", err.What, err.StoredConfig, err.NewConfig, err.RewindTo)
}

// Rules wraps ChainConfig and is merely syntactic sugar or can be used for functions
// that do not have or require information about the block.
//
// Rules is a one time interface meaning that it shouldn't be used in between transition
// phases.
type Rules struct {
	ChainID                                   *big.Int
	IsHomestead, IsEIP150, IsEIP155, IsEIP158 bool
	IsByzantium, IsConstantinople             bool
	IsPohang                                  bool
	IsUlsan                                   bool
}

// Rules ensures c's ChainID is not nil.
func (c *ChainConfig) Rules(num *big.Int) Rules {
	chainID := c.ChainID
	if chainID == nil {
		chainID = new(big.Int)
	}

	return Rules{
		ChainID:          new(big.Int).Set(chainID),
		IsHomestead:      c.IsHomestead(num),
		IsEIP150:         c.IsEIP150(num),
		IsEIP155:         c.IsEIP155(num),
		IsEIP158:         c.IsEIP158(num),
		IsByzantium:      c.IsByzantium(num),
		IsConstantinople: c.IsConstantinople(num),
		IsPohang:         c.IsPohang(num),
		IsUlsan:          c.IsUlsan(num),
	}

}

// TrustedCheckpoint represents a set of post-processed trie roots (CHT and
// BloomTrie) associated with the appropriate section index and head hash. It is
// used to start light syncing from this checkpoint and avoid downloading the
// entire header chain while still being able to securely access old headers/logs.
type TrustedCheckpoint struct {
	SectionIndex uint64      `json:"sectionIndex"`
	SectionHead  common.Hash `json:"sectionHead"`
	CHTRoot      common.Hash `json:"chtRoot"`
	BloomRoot    common.Hash `json:"bloomRoot"`
}

// HashEqual returns an indicator comparing the itself hash with given one.
func (c *TrustedCheckpoint) HashEqual(hash common.Hash) bool {
	if c.Empty() {
		return hash == common.Hash{}
	}
	return c.Hash() == hash
}

// Hash returns the hash of checkpoint's four key fields(index, sectionHead, chtRoot and bloomTrieRoot).
func (c *TrustedCheckpoint) Hash() common.Hash {
	var sectionIndex [8]byte
	binary.BigEndian.PutUint64(sectionIndex[:], c.SectionIndex)

	w := sha3.NewLegacyKeccak256()
	w.Write(sectionIndex[:])
	w.Write(c.SectionHead[:])
	w.Write(c.CHTRoot[:])
	w.Write(c.BloomRoot[:])

	var h common.Hash
	w.Sum(h[:0])
	return h
}

// Empty returns an indicator whether the checkpoint is regarded as empty.
func (c *TrustedCheckpoint) Empty() bool {
	return c.SectionHead == (common.Hash{}) || c.CHTRoot == (common.Hash{}) || c.BloomRoot == (common.Hash{})
}
