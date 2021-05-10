package fairdb

import (
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/params"
	"math/big"
	"sync"
)

type MemDatabase struct {
	mu sync.Mutex

	CurBlock *types.Block
	CurOtprn *types.Otprn

	ConfigList map[common.Hash]types.ChainConfig

	NodeList   map[string]types.HeartBeat
	OtprnList  []types.Otprn
	LeagueList map[common.Hash]map[string]types.HeartBeat
	VoteList   map[common.Hash][]*types.Voter
	BlockChain []*types.Block
}

func NewMemDatabase() *MemDatabase {
	return &MemDatabase{
		NodeList:   make(map[string]types.HeartBeat),
		LeagueList: make(map[common.Hash]map[string]types.HeartBeat),
		VoteList:   make(map[common.Hash][]*types.Voter),
		ConfigList: make(map[common.Hash]types.ChainConfig),
	}
}

func (m *MemDatabase) Start() error {
	logger.Debug("Start fairnode memory database")
	return nil
}
func (m *MemDatabase) Stop() {
	logger.Debug("Stop fairnode memory database")
}

func (m *MemDatabase) GetChainConfig() *types.ChainConfig {
	// sample
	return &types.ChainConfig{
		MinMiner:    3,
		Epoch:       10,
		Mminer:      50,
		FnFee:       big.NewFloat(10).String(),
		NodeVersion: params.Version,
		Price: types.Price{
			JoinTxPrice: "1",
			GasPrice:    params.MinimumGenesisGasPrice,
			GasLimit:    params.GenesisGasLimit,
		},
	}
}

func (m *MemDatabase) SaveChainConfig(config *types.ChainConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.ConfigList[config.Hash()]; ok {
		return errors.New(fmt.Sprintf("config exsist Hash = %s", config.Hash().String()))
	} else {
		m.ConfigList[config.Hash()] = *config
		return nil
	}
}

func (m *MemDatabase) CurrentInfo() *types.CurrentInfo {
	if len(m.BlockChain) == 0 {
		return nil
	}

	block := m.BlockChain[len(m.BlockChain)-1] // current block
	return &types.CurrentInfo{
		Number: block.Number(),
		Hash:   block.Hash(),
	}
}

func (m *MemDatabase) CurrentBlock() *types.Block {
	if len(m.BlockChain) == 0 {
		return nil
	}
	return m.BlockChain[len(m.BlockChain)-1] // current block
}

func (m *MemDatabase) CurrentOtprn() *types.Otprn {
	if len(m.OtprnList) == 0 {
		return nil
	}
	return &m.OtprnList[len(m.OtprnList)-1] // current Otprn
}

func (m *MemDatabase) InitActiveNode() {}

func (m *MemDatabase) SaveActiveNode(node types.HeartBeat) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.NodeList[node.Enode] = node
}

func (m *MemDatabase) GetActiveNode() []types.HeartBeat {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []types.HeartBeat
	for _, node := range m.NodeList {
		res = append(res, node)
	}
	return res
}

func (m *MemDatabase) RemoveActiveNode(enode string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if node, ok := m.NodeList[enode]; ok {
		delete(m.NodeList, node.Enode)
	}
}

func (m *MemDatabase) SaveOtprn(otprn types.Otprn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.OtprnList = append(m.OtprnList, otprn)
}

func (m *MemDatabase) GetOtprn(otprnHash common.Hash) *types.Otprn {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, otprn := range m.OtprnList {
		if otprn.HashOtprn() == otprnHash {
			return &otprn
		}
	}
	return nil
}

func (m *MemDatabase) SaveLeague(otprnHash common.Hash, enode string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if list, ok := m.LeagueList[otprnHash]; ok {
		if _, ok := list[enode]; !ok {
			m.LeagueList[otprnHash][enode] = m.NodeList[enode]
		}
	} else {
		m.LeagueList[otprnHash] = make(map[string]types.HeartBeat)
		m.LeagueList[otprnHash][enode] = m.NodeList[enode]
	}
}

func (m *MemDatabase) GetLeagueList(otprnHash common.Hash) []types.HeartBeat {
	m.mu.Lock()
	defer m.mu.Unlock()
	var leagues []types.HeartBeat
	if list, ok := m.LeagueList[otprnHash]; ok {
		for _, node := range list {
			leagues = append(leagues, node)
		}
	}
	return leagues
}

func (m *MemDatabase) SaveVote(otprn common.Hash, blockNum *big.Int, vote *types.Voter) {
	m.mu.Lock()
	defer m.mu.Unlock()

	voteKey := MakeVoteKey(otprn, blockNum)
	if list, ok := m.VoteList[voteKey]; ok {
		list = append(list, vote)
		m.VoteList[voteKey] = list
	} else {
		m.VoteList[voteKey] = []*types.Voter{vote}
	}
}

func (m *MemDatabase) GetVoters(votekey common.Hash) []*types.Voter {
	m.mu.Lock()
	defer m.mu.Unlock()
	if list, ok := m.VoteList[votekey]; ok {
		return list
	}
	return nil
}

func (m *MemDatabase) SaveFinalBlock(block *types.Block, byteBlock []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, b := range m.BlockChain {
		if b.Hash() == block.Hash() {
			return ErrAlreadyExistBlock
		}
	}

	m.BlockChain = append(m.BlockChain, block)
	return nil
}

func (m *MemDatabase) RemoveBlock(blockHash common.Hash) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := m.BlockChain
	for i, block := range c {
		if block.Hash() == blockHash {
			c[len(c)-1], c[i] = c[i], c[len(c)-1]
			m.BlockChain = c[:len(c)-1]
			return
		}
	}
}

func (m *MemDatabase) GetBlock(blockHash common.Hash) *types.Block {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, block := range m.BlockChain {
		if block.Hash() == blockHash {
			return block
		}
	}
	return nil
}

func (m *MemDatabase) DeletedBlockChangeState(blockHash common.Hash) {
	// ....
}