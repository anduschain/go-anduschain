package pool

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/server/db"
	"sync"
)

type VoteBlock struct {
	Block    *types.Block
	Count    uint64
	Voters   []common.Address
	Receipts []*types.Receipt
}

type VoteBlocks []VoteBlock

type Vote struct {
	Hash     OtprnHash
	Block    *types.Block
	Coinbase common.Address
	Receipts []*types.Receipt
}

type VotePool struct {
	pool     map[OtprnHash]VoteBlocks
	InsertCh chan Vote
	SnapShot chan *types.Block
	DeleteCh chan OtprnHash
	StopCh   chan struct{}
	db       *db.FairNodeDB
	mux      sync.RWMutex
}

func NewVotePool(db *db.FairNodeDB) *VotePool {
	vp := &VotePool{
		pool:     make(map[OtprnHash]VoteBlocks),
		InsertCh: make(chan Vote),
		SnapShot: make(chan *types.Block),
		StopCh:   make(chan struct{}, 1),
		DeleteCh: make(chan OtprnHash),
		db:       db,
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
			if val, ok := vp.pool[vote.Hash]; ok {
				isDouble := false

			ex:
				for i := range val {
					// 중복 삽입 방지
					if val[i].Block.Hash() == vote.Block.Hash() {
						// 동일한 블록이면, 카운터 +1, 투표자 추가

						for j := range val[i].Voters {
							if val[i].Voters[j] == vote.Coinbase {
								isDouble = true
								break ex
							}
						}

						vp.mux.Lock()
						vp.pool[vote.Hash][i].Count++
						vp.pool[vote.Hash][i].Voters = append(vp.pool[vote.Hash][i].Voters, vote.Coinbase)
						vp.mux.Unlock()

						isDouble = true
						break
					}
				}

				if !isDouble {
					vp.mux.Lock()
					vp.pool[vote.Hash] = append(val, VoteBlock{vote.Block, 1, []common.Address{vote.Coinbase}, vote.Receipts})
					vp.mux.Unlock()
				}

			} else {
				vp.mux.Lock()
				vp.pool[vote.Hash] = VoteBlocks{VoteBlock{vote.Block, 1, []common.Address{vote.Coinbase}, vote.Receipts}}
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
		}
	}
}
