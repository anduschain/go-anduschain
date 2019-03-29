package deb

import (
	"errors"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/consensus"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/crypto"
	"github.com/anduschain/go-anduschain/fairnode/client/config"
	types2 "github.com/anduschain/go-anduschain/fairnode/client/types"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/fairutil"
	"github.com/anduschain/go-anduschain/rlp"
	"math/big"
)

const (
	VotingWaitTime = 10 // 리그에서 블록 투표시 기다리는 시간
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
		if fairutil.CmpAddress(addr, c.fairAddr) {
			return nil, -1
		} else {
			return errNotMatchFairAddress, ErrNotMatchFairAddress
		}

	} else {
		return errNonFairNodeSig, ErrNonFairNodeSig
	}
}

// 투표블록 서명 검증하고 난이도 검증
func (c *Deb) ValidateJointx(voteBlock *fairtypes.VoteBlock) bool {

	return false
}

func (c *Deb) ValidationVoteBlock(chain consensus.ChainReader, voteblock *types.Block) error {
	if chain.CurrentHeader().Number.Uint64()+1 != voteblock.Number().Uint64() {
		return errNotMatchOtprnOrBlockNumber
	}
	// check otprn
	if c.otprnHash != common.BytesToHash(voteblock.Extra()) {
		return errNotMatchOtprnOrBlockNumber
	}

	//header 검증
	err := c.verifyHeader(chain, voteblock.Header(), nil)
	if err != nil {
		return err
	}

	// Ensure the timestamp has the correct delay
	parent := chain.GetHeader(voteblock.ParentHash(), voteblock.Number().Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}

	current, err := chain.StateAt(parent.Root)
	if err != nil {
		return err
	}
	// join tx check
	err = c.ValidationBlockWidthJoinTx(chain.Config().ChainID, voteblock, current.GetJoinNonce(voteblock.Coinbase()))
	if err != nil {
		return err
	}

	return nil
}

func (c *Deb) ValidationBlockWidthJoinTx(chainid *big.Int, block *types.Block, joinNonce uint64) error {
	signer := types.NewEIP155Signer(chainid)
	txs := block.Transactions()
	var datas types2.JoinTxData
	var isMyJoinTx bool
	for i := range txs {
		if txs[i].To() != nil {
			if fairutil.CmpAddress(*txs[i].To(), c.fairAddr) {
				err := rlp.DecodeBytes(txs[i].Data(), &datas)
				if err != nil {
					return errDecodeTx
				}
				//참가비확인
				if txs[i].Value().Cmp(config.Price) != 0 {
					return errTxTicketPriceNotAvailable
				}

				//내 jointx가 있는지 확인 && otprn
				if c.otprnHash == datas.OtprnHash && datas.NextBlockNum == block.Number().Uint64() {
					from, _ := types.Sender(signer, txs[i])
					if fairutil.CmpAddress(from, block.Header().Coinbase) {
						if datas.JoinNonce == joinNonce {
							isMyJoinTx = true
						} else {
							c.logger.Debug("JOIN_NONCE", "joinTx", datas.JoinNonce, "current", joinNonce)
							return errors.New("JoinNonce가 다르다")
						}
					}
				}
			}
		}
	}

	if isMyJoinTx {
		return nil
	}

	return errNotInJoinTX
}

// 투표블록 서명 검증하고 난이도 검증
func (c *Deb) ValidationVoteBlockSign(voteBlock *fairtypes.VoteBlock) bool {
	block := voteBlock.Block
	header := voteBlock.Block.Header()

	pubKey, err := crypto.SigToPub(header.Hash().Bytes(), voteBlock.Sig)
	if err != nil {
		return false
	}

	addr := crypto.PubkeyToAddress(*pubKey)
	if block.Header().Coinbase.String() == addr.String() {
		return true
	} else {
		return false
	}
}

// 다른데서 받은 투표 블록을 비교하여 위닝 블록으로 교체 하는 함수
func (c *Deb) CompareBlock(myBlock, receivedBlock *fairtypes.VoteBlock) *fairtypes.VoteBlock {

	pvBlock := myBlock // 가지고 있던 블록
	voteBlocks := receivedBlock

	if pvBlock == nil {
		return voteBlocks
	}

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

func (c *Deb) SendMiningBlockAndVoting(chain consensus.ChainReader, tsfBlock *fairtypes.VoteBlock, isVoting *bool) {
	c.logger.Debug("SendMiningBlockAndVoting Run", "otprnHash", tsfBlock.OtprnHash.String(), "blockNum", tsfBlock.Block.Number())
	var winningBlock *fairtypes.VoteBlock
	prevHash := tsfBlock.Block.Hash()
	//t := time.NewTicker(VotingWaitTime * time.Second)

	defer func() {
		*isVoting = false
		c.logger.Debug("SendMiningBlockAndVoting Closed", "otprnHash", tsfBlock.OtprnHash.String(), "blockNum", tsfBlock.Block.Number())
	}()

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
					err := c.ValidationVoteBlock(chain, recevedBlock.Block)
					if err != nil {
						continue
					}

					if !c.ValidationVoteBlockSign(recevedBlock) {
						continue
					}

					c.logger.Debug("리그 전파 블록 도착", "blockNum", recevedBlock.Block.Number(), "hash", recevedBlock.Block.Header().Hash().String())

					// winningblock을 선정하고, 우선순위가 높은 블록을 다시 재배포
					winningBlock = c.CompareBlock(winningBlock, recevedBlock)

					if prevHash != winningBlock.Block.Hash() {
						block := winningBlock.Block
						sig, err := c.SignBlockHeader(block.Header().Hash().Bytes())
						if err != nil {
							c.logger.Error("Boradcasting Block found but fail signature", "number", block.Number(), "hash", block.Hash())
							continue
						}

						c.chans.GetLeagueBlockBroadcastCh() <- &fairtypes.VoteBlock{
							Block:      block,
							HeaderHash: block.Header().Hash(),
							Sig:        sig,
							OtprnHash:  winningBlock.OtprnHash,
							Voter:      c.coinbase,
						}

						prevHash = winningBlock.Block.Hash()
					}
				}
			}
		case <-c.chans.GetWinningBlockVoteStartCh():
			wb := winningBlock

			if wb == nil {
				c.logger.Debug("블록투표 실패", "blockNum", tsfBlock.Block.Header().Number, "hash", tsfBlock.Block.Header().Hash())
				break Exit
			}

			if chain.CurrentHeader().Number.Cmp(wb.Block.Number()) >= 0 {
				continue
			}
			// 위닝블록 전송
			mySig, err := c.SignBlockHeader(wb.Block.Header().Hash().Bytes())
			if err != nil {
				continue
			}

			c.chans.GetWinningBlockCh() <- &fairtypes.Vote{
				BlockNum:   wb.Block.Header().Number,
				HeaderHash: wb.Block.Header().Hash(),
				Sig:        mySig,
				Voter:      c.coinbase,
				OtprnHash:  wb.OtprnHash,
				Difficulty: c.difficulty,
			}

			c.client.SaveWiningBlock(wb.OtprnHash, wb.Block)
			c.logger.Debug("블록투표 ", "blockNum", wb.Block.Header().Number, "hash", wb.Block.Header().Hash())
			break Exit
		}
	}
}
