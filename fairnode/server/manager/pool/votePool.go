package pool

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/server/db"
	"sync"
)

type ReqBlock struct {
	Addr      common.Address
	BlockHash BlockHash
}

type BlockHash common.Hash

type VoteBlock struct {
	BlockHash BlockHash
	Voters    []types.Voter
	Count     uint64
}

type VoteBlocks []VoteBlock

type Vote struct {
	OtprnHash  OtprnHash
	HeaderHash common.Hash
	types.Voter
}

type VotePool struct {
	pool           map[OtprnHash]VoteBlocks
	voteBlocks     map[OtprnHash]map[BlockHash]*types.Block
	InsertCh       chan Vote
	SnapShot       chan *types.Block
	DeleteCh       chan OtprnHash
	StopCh         chan struct{}
	RequestBlockCh chan ReqBlock
	StoreBlockCh   chan *fairtypes.ResWinningBlock
	db             *db.FairNodeDB
	mux            sync.RWMutex
}

func NewVotePool(db *db.FairNodeDB) *VotePool {
	vp := &VotePool{
		pool:           make(map[OtprnHash]VoteBlocks),
		voteBlocks:     make(map[OtprnHash]map[BlockHash]*types.Block),
		InsertCh:       make(chan Vote),
		SnapShot:       make(chan *types.Block),
		StopCh:         make(chan struct{}, 1),
		DeleteCh:       make(chan OtprnHash),
		RequestBlockCh: make(chan ReqBlock),
		StoreBlockCh:   make(chan *fairtypes.ResWinningBlock),
		db:             db,
	}

	return vp
}

func (vp *VotePool) GetVoteBlocks(hash OtprnHash) VoteBlocks {
	vp.mux.Lock()
	defer vp.mux.Unlock()
	if val, ok := vp.pool[hash]; ok {
		return val
	}

	return nil
}

func (vp *VotePool) GetBlock(hash OtprnHash, blockhash BlockHash) *types.Block {
	vp.mux.Lock()
	defer vp.mux.Unlock()

	if block, ok := vp.voteBlocks[hash][blockhash]; ok {
		return block
	}

	return nil
}

func (vp *VotePool) Start() error {
	go vp.loop()
	return nil
}

func (vp *VotePool) Stop() error {
	vp.StopCh <- struct{}{}
	return nil
}

func (vp *VotePool) loop() {

Exit:
	for {
		select {
		case vote := <-vp.InsertCh:

			if val, ok := vp.pool[vote.OtprnHash]; ok {
				isDouble := false
			ex:
				for i := range val {
					// 중복 삽입 방지
					if val[i].BlockHash == BlockHash(vote.HeaderHash) {
						// 동일한 블록헤더이면, 카운터 +1, 투표자 추가

						for j := range val[i].Voters {
							if val[i].Voters[j].Addr == vote.Addr {
								isDouble = true
								break ex
							}
						}

						vp.mux.Lock()
						vp.pool[vote.OtprnHash][i].Count++
						vp.pool[vote.OtprnHash][i].Voters = append(vp.pool[vote.OtprnHash][i].Voters, types.Voter{vote.Addr, vote.Sig})
						vp.mux.Unlock()

						isDouble = true
						break
					}
				}

				if !isDouble {
					vp.mux.Lock()
					vp.pool[vote.OtprnHash] = append(val, VoteBlock{BlockHash(vote.HeaderHash), []types.Voter{{vote.Addr, vote.Sig}}, 1})
					// 블록을 요청함
					vp.RequestBlockCh <- ReqBlock{vote.Addr, BlockHash(vote.HeaderHash)}
					vp.mux.Unlock()
				}

			} else {
				vp.mux.Lock()
				vp.pool[vote.OtprnHash] = VoteBlocks{VoteBlock{BlockHash(vote.HeaderHash), []types.Voter{{vote.Addr, vote.Sig}}, 1}}
				// 블록을 요청함
				vp.RequestBlockCh <- ReqBlock{vote.Addr, BlockHash(vote.HeaderHash)}
				vp.mux.Unlock()
			}
		case block := <-vp.SnapShot:
			vp.db.SaveFianlBlock(block)
		case h := <-vp.DeleteCh:
			if _, ok := vp.pool[h]; ok {
				vp.mux.Lock()
				delete(vp.pool, h)
				vp.mux.Unlock()
			}
		case <-vp.StopCh:
			break Exit
		case resBlock := <-vp.StoreBlockCh:
			vp.mux.Lock()
			if val, ok := vp.voteBlocks[OtprnHash(resBlock.OtprnHash.String())]; ok {
				val[BlockHash(resBlock.Block.Header().Hash())] = resBlock.Block
			} else {
				vp.voteBlocks[OtprnHash(resBlock.OtprnHash.String())] = make(map[BlockHash]*types.Block)
				vp.voteBlocks[OtprnHash(resBlock.OtprnHash.String())][BlockHash(resBlock.Block.Header().Hash())] = resBlock.Block
			}
			vp.mux.Unlock()
		}
	}
}
