package fairtcp

import (
	"fmt"
	"github.com/anduschain/go-anduschain/accounts"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/fairutil"
	"github.com/anduschain/go-anduschain/fairnode/server/manager/pool"
	"github.com/anduschain/go-anduschain/fairnode/transport"
	"log"
	"time"
)

func (fu *FairTcp) sendLeague(otprnHash string) {
	t := time.NewTicker(5 * time.Second)
	var percent float64 = 30

	for {
		select {
		case <-t.C:
			_, num, enodes := fu.leaguePool.GetLeagueList(pool.StringToOtprn(otprnHash))
			// 가능한 사람의 30%이상일때 접속할 채굴 리그를 전송해줌
			if num >= fu.JoinTotalNum(percent) && num > 0 {

				fmt.Println("-------리그 전송---------")
				fu.sendTcpAll(transport.SendLeageNodeList, enodes)

				time.AfterFunc(5*time.Second, func() {
					fmt.Println("-------조인 tx 생성--------")
					fu.sendTcpAll(transport.MakeJoinTx, otprnHash)

					time.AfterFunc(5*time.Second, func() {
						fmt.Println("-------블록 생성--------", otprnHash)
						fu.sendTcpAll(transport.MakeBlock, otprnHash)

						// peer list 전송후 20초
						time.AfterFunc(20*time.Second, func() {
							go fu.sendFinalBlock(otprnHash)
						})
					})
				})

				return
			}
		}
	}
}

func (fu *FairTcp) sendTcpAll(msgCode uint32, data interface{}) {
	otprnHash := fu.manager.GetOtprn().HashOtprn().String()
	nodes, _, _ := fu.leaguePool.GetLeagueList(pool.StringToOtprn(otprnHash))
	for index := range nodes {
		if nodes[index].Conn != nil {
			transport.Send(nodes[index].Conn, msgCode, data)
		}
	}
}

func (fu *FairTcp) sendFinalBlock(otprnHash string) {
	leaguePool := fu.manager.GetLeaguePool()
	votePool := fu.manager.GetVotePool()
	notify := make(chan *fairtypes.FinalBlock)

	go func() {
		t := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-t.C:
				fb := fu.GetFinalBlock(otprnHash, votePool)
				if fb == nil {
					continue
				} else {
					notify <- fb
					return
				}
			}
		}
	}()

	for {
		select {
		case n := <-notify:
			fu.sendTcpAll(transport.SendFinalBlock, n.GetTsFinalBlock())
			fmt.Println("----파이널 블록 전송-----", n.Block.NumberU64(), n.Block.Coinbase().String())

			leaguePool.SnapShot <- pool.StringToOtprn(otprnHash)
			leaguePool.DeleteCh <- pool.StringToOtprn(otprnHash)

			// DB에 블록 저장
			votePool.SnapShot <- n.Block
			votePool.DeleteCh <- pool.StringToOtprn(otprnHash)

			fu.StopLeague()
			return
		}
	}
}

func (fu *FairTcp) JoinTotalNum(persent float64) uint64 {
	aciveNode := fu.Db.GetActiveNodeList()
	var count float64 = 0
	for i := range aciveNode {
		if fairutil.IsJoinOK(fu.manager.GetOtprn(), common.HexToAddress(aciveNode[i].Coinbase)) {
			count += 1
		}
	}

	return uint64(count * (persent / 100))
}

func (fu *FairTcp) GetFinalBlock(otprnHash string, votePool *pool.VotePool) *fairtypes.FinalBlock {
	voteBlocks := votePool.GetVoteBlocks(pool.StringToOtprn(otprnHash))
	acc := fu.manager.GetServerKey()

	var fb fairtypes.FinalBlock

	if len(voteBlocks) == 0 {
		return nil
	} else if len(voteBlocks) == 1 {
		fmt.Println("--------------count == 1----------")
		fb.Block = voteBlocks[0].Block
		//fb.Receipts = voteBlocks[0].Receipts
		SignFairNode(fb.Block, voteBlocks[0], acc.ServerAcc, acc.KeyStore)
	} else {
		var cnt uint64 = 0
		var pvBlock pool.VoteBlock
		for i := range voteBlocks {
			// 1. count가 높은 블록
			// 2. Rand == diffcult 값이 높은 블록
			// 3. joinNunce	== nonce 값이 놓은 블록
			// 4. 블록이 홀수 이면 - 주소값이 작은사람 , 블록이 짝수이면 - 주소값이 큰사람
			if cnt < voteBlocks[i].Count {
				fb.Block = voteBlocks[i].Block
				//fb.Receipts = voteBlocks[i].Receipts
				pvBlock = voteBlocks[i]
				cnt = voteBlocks[i].Count
			} else if cnt == voteBlocks[i].Count {
				// 동수인 투표일때
				if voteBlocks[i].Block.Difficulty().Cmp(pvBlock.Block.Difficulty()) == 1 {
					// diffcult 값이 높은 블록
					fb.Block = voteBlocks[i].Block
					//fb.Receipts = voteBlocks[i].Receipts
					pvBlock = voteBlocks[i]
				} else if voteBlocks[i].Block.Difficulty().Cmp(pvBlock.Block.Difficulty()) == 0 {
					// diffcult 값이 같을때
					if voteBlocks[i].Block.Nonce() > pvBlock.Block.Nonce() {
						// nonce 값이 큰 블록
						fb.Block = voteBlocks[i].Block
						//fb.Receipts = voteBlocks[i].Receipts
						pvBlock = voteBlocks[i]
					} else if voteBlocks[i].Block.Nonce() == pvBlock.Block.Nonce() {
						// nonce 값이 같을 때
						if voteBlocks[i].Block.Number().Uint64()%2 == 0 {
							// 블록 번호가 짝수 일때
							if voteBlocks[i].Block.Coinbase().Big().Cmp(pvBlock.Block.Coinbase().Big()) == 1 {
								// 주소값이 큰 블록
								fb.Block = voteBlocks[i].Block
								//fb.Receipts = voteBlocks[i].Receipts
								pvBlock = voteBlocks[i]
							}
						} else {
							// 블록 번호가 홀수 일때
							if voteBlocks[i].Block.Coinbase().Big().Cmp(pvBlock.Block.Coinbase().Big()) == -1 {
								// 주소값이 작은 블록
								fb.Block = voteBlocks[i].Block
								//fb.Receipts = voteBlocks[i].Receipts
								pvBlock = voteBlocks[i]
							}
						}

					}
				}
			}
		}

		SignFairNode(fb.Block, pvBlock, acc.ServerAcc, acc.KeyStore)
	}

	return &fb
}

func SignFairNode(block *types.Block, vBlock pool.VoteBlock, account accounts.Account, ks *keystore.KeyStore) {
	sig, err := ks.SignHash(account, vBlock.Block.Hash().Bytes())
	if err != nil {
		log.Println("Error[andus] : SignFairNode 서명에러", err)
	}

	block.Voter = vBlock.Voters
	block.FairNodeSig = sig
}
