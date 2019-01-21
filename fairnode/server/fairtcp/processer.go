package fairtcp

import (
	"fmt"
	"github.com/anduschain/go-anduschain/accounts"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/fairutil"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/fairnode/server/manager/pool"
	"github.com/anduschain/go-anduschain/fairnode/transport"
	"log"
	"time"
)

func (fu *FairTcp) sendLeague(otprnHash common.Hash) {
	t := time.NewTicker(5 * time.Second)
	var percent float64 = 30

	for {
		select {
		case <-t.C:
			if fu.manager.GetUsingOtprn() == nil {
				return
			}
			if otprnHash != fu.manager.GetUsingOtprn().HashOtprn() {
				return
			}
			_, num, enodes := fu.leaguePool.GetLeagueList(pool.OtprnHash(otprnHash))
			// 가능한 사람의 30%이상일때 접속할 채굴 리그를 전송해줌
			if fu.manager.GetUsingOtprn() == nil {
				continue
			}
			if num >= fu.JoinTotalNum(fu.manager.GetUsingOtprn(), percent) && num > 0 {

				fmt.Println("-------리그 전송---------")
				fu.sendTcpAll(otprnHash, transport.SendLeageNodeList, enodes)
				go fu.leagueControlle(otprnHash)
				time.Sleep(3 * time.Second)
				fu.makeJoinTxCh <- struct{}{}

				return
			}

			if percent > 5 {
				// 최소 5%
				percent = percent - 5
			}
		}
	}
}

func (fu *FairTcp) leagueControlle(otprnHash common.Hash) {
	for {
		select {
		case <-fu.makeJoinTxCh:
			fmt.Println("-------조인 tx 생성--------")
			fu.sendTcpAll(otprnHash, transport.MakeJoinTx, fairtypes.BlockMakeMessage{otprnHash, fu.manager.GetLastBlockNum().Uint64() + 1})
			// 브로드케스팅 5초
			time.AfterFunc(3*time.Second, func() {

				fmt.Println("-------블록 생성--------", otprnHash.String(), fu.manager.GetLastBlockNum().Uint64()+1)
				fu.sendTcpAll(otprnHash, transport.MakeBlock, fairtypes.BlockMakeMessage{otprnHash, fu.manager.GetLastBlockNum().Uint64() + 1})

				// peer list 전송후 20초
				time.AfterFunc(10*time.Second, func() {
					go fu.sendFinalBlock(otprnHash)
				})

			})
		case <-fu.manager.GetStopLeagueCh():
			fu.StopLeague(otprnHash)
			leaguePool := fu.manager.GetLeaguePool()
			leaguePool.SnapShot <- pool.OtprnHash(otprnHash)
			leaguePool.DeleteCh <- pool.OtprnHash(otprnHash)
			return
		}
	}
}

func (fu *FairTcp) sendTcpAll(otprnHash common.Hash, msgCode uint32, data interface{}) {
	nodes, _, _ := fu.leaguePool.GetLeagueList(pool.OtprnHash(otprnHash))
	for index := range nodes {
		if nodes[index].Conn != nil {
			err := transport.Send(nodes[index].Conn, msgCode, data)
			if err != nil {
				// 데이터 전송 오류 시 해당 연결 삭제
				nodes[index].Conn = nil
			}
		}
	}
}

func (fu *FairTcp) sendFinalBlock(otprnHash common.Hash) {
	votePool := fu.manager.GetVotePool()
	notify := make(chan *fairtypes.FinalBlock)
	defer fmt.Println("sendFinalBlock 죽음")
	go func() {
		t := time.NewTicker(1 * time.Second)
		conter := 0
		for {
			select {
			case <-t.C:
				fb := fu.GetFinalBlock(otprnHash, votePool)
				if fb == nil {
					//5초 이후에 리그 교체
					if conter == 4 {
						notify <- nil
						return
					}
					conter++
					fmt.Println(conter)
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
			if n != nil {
				fu.sendTcpAll(otprnHash, transport.SendFinalBlock, n.GetTsFinalBlock())
				fmt.Println("----파이널 블록 전송-----", n.Block.NumberU64(), n.Block.Coinbase().String())

				// DB에 블록 저장
				votePool.SnapShot <- n.Block
				votePool.DeleteCh <- pool.OtprnHash(otprnHash)

				time.Sleep(5 * time.Second)
				fu.makeJoinTxCh <- struct{}{}
				fu.manager.GetManagerOtprnCh() <- struct{}{}
				return
			} else {
				fmt.Println("----파이널 블록 전송 시간 초과로 인한 리그 교체--------")
				fu.manager.GetStopLeagueCh() <- struct{}{}
				return
			}
		}
	}
}

func (fu *FairTcp) JoinTotalNum(otprn *otprn.Otprn, persent float64) uint64 {
	aciveNode := fu.Db.GetActiveNodeList()
	var count float64 = 0
	for i := range aciveNode {
		if fairutil.IsJoinOK(otprn, common.HexToAddress(aciveNode[i].Coinbase)) {
			count += 1
		}
	}

	return uint64(count * (persent / 100))
}

func (fu *FairTcp) GetFinalBlock(otprnHash common.Hash, votePool *pool.VotePool) *fairtypes.FinalBlock {
	otrpnHash := pool.OtprnHash(otprnHash)
	votes := votePool.GetVoteBlocks(otrpnHash)
	acc := fu.manager.GetServerKey()
	fb := &fairtypes.FinalBlock{}

	type vB struct {
		Block *types.Block
		Voter pool.VoteBlock
	}
	var voteBlocks []vB
	for i := range votes {
		block := votePool.GetBlock(otrpnHash, votes[i].BlockHash)
		if block == nil {
			continue
		}
		voteBlocks = append(voteBlocks, vB{block, votes[i]})
	}

	if len(voteBlocks) == 0 {
		return nil
	} else if len(voteBlocks) == 1 {
		fb.Block = voteBlocks[0].Block
		SignFairNode(fb.Block, voteBlocks[0].Voter, acc.ServerAcc, acc.KeyStore)
	} else {
		var cnt uint64 = 0
		var pv vB
		for i := range voteBlocks {
			voter := voteBlocks[i]
			count := voteBlocks[i].Voter.Count
			block := voteBlocks[i].Block
			if block == nil {
				continue
			}
			// 1. count가 높은 블록
			// 2. Rand == diffcult 값이 높은 블록
			// 3. joinNunce	== nonce 값이 놓은 블록
			// 4. 블록이 홀수 이면 - 주소값이 작은사람 , 블록이 짝수이면 - 주소값이 큰사람
			if cnt < count {
				fb.Block = block
				pv = voter
				cnt = count
			} else if cnt == count {
				// 동수인 투표일때
				if voteBlocks[i].Block.Difficulty().Cmp(pv.Block.Difficulty()) == 1 {
					// diffcult 값이 높은 블록
					fb.Block = block
					pv = voter
				} else if voteBlocks[i].Block.Difficulty().Cmp(pv.Block.Difficulty()) == 0 {
					// diffcult 값이 같을때
					if voteBlocks[i].Block.Nonce() > pv.Block.Nonce() {
						// nonce 값이 큰 블록
						fb.Block = block
						pv = voter
					} else if voteBlocks[i].Block.Nonce() == pv.Block.Nonce() {
						// nonce 값이 같을 때
						if voteBlocks[i].Block.Number().Uint64()%2 == 0 {
							// 블록 번호가 짝수 일때
							if voteBlocks[i].Block.Coinbase().Big().Cmp(pv.Block.Coinbase().Big()) == 1 {
								// 주소값이 큰 블록
								fb.Block = block
								pv = voter
							}
						} else {
							// 블록 번호가 홀수 일때
							if voteBlocks[i].Block.Coinbase().Big().Cmp(pv.Block.Coinbase().Big()) == -1 {
								// 주소값이 작은 블록
								fb.Block = block
								pv = voter
							}
						}

					}
				}
			}
		}

		SignFairNode(fb.Block, pv.Voter, acc.ServerAcc, acc.KeyStore)
	}

	return fb
}

func SignFairNode(block *types.Block, vBlock pool.VoteBlock, account accounts.Account, ks *keystore.KeyStore) {
	sig, err := ks.SignHash(account, block.Hash().Bytes())
	if err != nil {
		log.Println("Error[andus] : SignFairNode 서명에러", err)
	}

	block.Voter = vBlock.Voters
	block.FairNodeSig = sig
}
