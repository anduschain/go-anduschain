package fairdb

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"math/big"
	"sync"
)

type MemDatabase struct {
	mu sync.Mutex

	CurBlock *types.Block
	CurOtprn *types.Otprn

	NodeList   map[string]types.HeartBeat
	OtprnList  []types.Otprn
	LeagueList map[common.Hash][]types.HeartBeat
	VoteList   map[common.Hash][]types.Voter
	BlockChain []types.Block
}

func NewMemDatabase() *MemDatabase {
	return &MemDatabase{
		NodeList:   make(map[string]types.HeartBeat),
		LeagueList: make(map[common.Hash][]types.HeartBeat),
		VoteList:   make(map[common.Hash][]types.Voter),
	}
}

func (m *MemDatabase) Start() error {
	logger.Debug("Start fairnode memory database")
	return nil
}
func (m *MemDatabase) Stop() {
	logger.Debug("Stop fairnode memory database")
}

func (m *MemDatabase) CurrentBlock() *types.Block {
	if len(m.BlockChain) == 0 {
		return nil
	}
	return &m.BlockChain[len(m.BlockChain)-1] // current block
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

func (m *MemDatabase) SaveLeagueList(otprnHash common.Hash, lists []types.HeartBeat) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.LeagueList[otprnHash] = lists
}

func (m *MemDatabase) GetLeagueList(otprnHash common.Hash) []types.HeartBeat {
	m.mu.Lock()
	defer m.mu.Unlock()
	if list, ok := m.LeagueList[otprnHash]; ok {
		return list
	}
	return nil
}

func (m *MemDatabase) SaveVote(otprn common.Hash, blockNum *big.Int, vote types.Voter) {
	m.mu.Lock()
	defer m.mu.Unlock()

	voteKey := makeVoteKey(otprn, blockNum)

	if list, ok := m.VoteList[voteKey]; ok {
		list = append(list, vote)
		m.VoteList[voteKey] = list
	} else {
		m.VoteList[voteKey] = []types.Voter{vote}
	}
}

func (m *MemDatabase) GetVoters(votekey common.Hash) []types.Voter {
	m.mu.Lock()
	defer m.mu.Unlock()
	if list, ok := m.VoteList[votekey]; ok {
		return list
	}
	return nil
}

func (m *MemDatabase) SaveFinalBlock(block types.Block) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.BlockChain = append(m.BlockChain, block)
}

func (m *MemDatabase) GetBlock(blockHash common.Hash) *types.Block {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, block := range m.BlockChain {
		if block.Hash() == blockHash {
			return &block
		}
	}
	return nil
}
