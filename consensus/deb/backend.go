package deb

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/consensus"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/fairnode/client/config"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"math/big"
	"time"
)

func (c *Deb) FairNodeSigCheck(recivedBlock *types.Block, rSig []byte) (error, ErrorType) {
	// TODO : andus >> FairNode의 서명이 있는지 확인 하고 검증
	sig := rSig

	if _, ok := recivedBlock.GetFairNodeSig(); ok {

		fpKey, err := crypto.SigToPub(recivedBlock.Header().Hash().Bytes(), sig)
		if err != nil {
			return errGetPubKeyError, ErrGetPubKeyError
		}

		addr := crypto.PubkeyToAddress(*fpKey)
		if addr.String() == config.FAIRNODE_ADDRESS {
			return nil, -1
		} else {
			return errNotMatchFairAddress, ErrNotMatchFairAddress
		}

	} else {
		return errNonFairNodeSig, ErrNonFairNodeSig
	}
}

func (c *Deb) ValidationVoteBlock(chain consensus.ChainReader, voteblock *types.Block) bool {
	if chain.CurrentHeader().Number.Uint64()+1 == voteblock.Number().Uint64() {
		if c.otprnHash == common.BytesToHash(voteblock.Extra()) {
			return true
		}
	}
	return false
}

func (c *Deb) CheckRANDSigOK(voteBlock *fairtypes.VoteBlock) bool {
	block := voteBlock.Block
	header := voteBlock.Block.Header()
	otprnHash := common.BytesToHash(voteBlock.Block.Header().Extra)
	rand := MakeRand(header.Nonce.Uint64(), otprnHash, header.Coinbase, header.ParentHash)
	diff := big.NewInt(rand)

	pubKey, err := crypto.SigToPub(header.Hash().Bytes(), voteBlock.Sig)
	if err != nil {
		return false
	}

	addr := crypto.PubkeyToAddress(*pubKey)
	if block.Header().Coinbase.String() == addr.String() {
		if header.Difficulty.Cmp(diff) == 0 {
			return true
		} else {
			return false
		}
	} else {
		return false
	}
}

// 다른데서 받은 투표 블록을 비교하여 위닝 블록으로 교체 하는 함수
func (c *Deb) CompareBlock(myBlock, receivedBlock *fairtypes.VoteBlock) *fairtypes.VoteBlock {

	pvBlock := myBlock // 가지고 있던 블록
	voteBlocks := receivedBlock

	if voteBlocks.Block.Difficulty().Cmp(pvBlock.Block.Difficulty()) == 1 {
		// diffcult 값이 높은 블록
		return voteBlocks
	} else if voteBlocks.Block.Difficulty().Cmp(pvBlock.Block.Difficulty()) == 0 {
		// diffcult 값이 같을때
		if voteBlocks.Block.Nonce() > pvBlock.Block.Nonce() {
			// nonce 값이 큰 블록
			return voteBlocks
		} else if voteBlocks.Block.Nonce() == pvBlock.Block.Nonce() {
			// nonce 값이 같을 때
			if voteBlocks.Block.Number().Uint64()%2 == 0 {
				// 블록 번호가 짝수 일때
				if voteBlocks.Block.Coinbase().Big().Cmp(pvBlock.Block.Coinbase().Big()) == 1 {
					// 주소값이 큰 블록
					return voteBlocks
				}
			} else {
				// 블록 번호가 홀수 일때
				if voteBlocks.Block.Coinbase().Big().Cmp(pvBlock.Block.Coinbase().Big()) == -1 {
					// 주소값이 작은 블록
					return pvBlock
				}
			}

		}
	} else {
		return pvBlock
	}

	return pvBlock
}

func (c *Deb) SendMiningBlockAndVoting(chain consensus.ChainReader, tsfBlock *fairtypes.VoteBlock) {
	winningBlock := tsfBlock
	t := time.NewTicker(10 * time.Second)

Exit:
	for {
		select {
		case recevedBlock := <-c.chans.GetReceiveBlockCh():
			// TODO : andus >> 블록 검증
			// TODO : andus >> 1. 받은 블록이 채굴리그 참여자가 생성했는지 여부를 확인
			if err, errType := c.FairNodeSigCheck(recevedBlock.Block, recevedBlock.Sig); err != nil {
				switch errType {
				case ErrNonFairNodeSig:
					// TODO : andus >> 2. RAND 값 서명 검증
					if c.ValidationVoteBlock(chain, recevedBlock.Block) {
						if OK := c.CheckRANDSigOK(recevedBlock); OK {
							winningBlock = c.CompareBlock(winningBlock, recevedBlock)
							fmt.Println("-------CheckRANDSigOK---winningBlock 교체-----")
						}
					}
				}
			}
		case <-t.C:
			// 위닝블록 전송
			c.chans.GetWinningBlockCh() <- winningBlock
			fmt.Println("-----------------FairNode로 winningBlock 전송-----------------")
			break Exit
		}
	}
}
