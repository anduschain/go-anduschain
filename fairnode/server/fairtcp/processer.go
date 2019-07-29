package fairtcp

import (
	"container/ring"
	"github.com/anduschain/go-anduschain/accounts"
	"github.com/anduschain/go-anduschain/accounts/keystore"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/fairutil"
	"github.com/anduschain/go-anduschain/fairnode/server/manager/pool"
	"github.com/anduschain/go-anduschain/fairnode/transport"
	"time"
)

const (
	sendLeagueCounter = 5  // 리그 리스트를 보낼시 리그 풀에 노드 갯수가 리그 참여가능한 갯수보다 LeaguePercent 보다 적을때 다시 계산 하는 시간
	LeaguePercent     = 30 // 초기 sendLeague 가능한 percent
	reducePercent     = 5  // 한번 다시 계산할때마다 줄어드는 percent

	makeJoinTxSig = 3 // 리그 리스트 전송 후 joinTx를 만들기까지의 대기시간

	finalBlockSig      = 10 // geth노드에게 블록 생성 메세지를 본낸뒤 geth노드들의 투표를 받고 확정된 블록을 보내주기까지의 시간
	noVoteBolckLeagChn = 10 // 투표받은 블록이 없어 다음 리그를 시작할 시간(조회 횟수)
	nextBlockMakeTerm  = 7  // 블록생성 후  다음 블록 생성 신호 전달 term
	blockVote          = 10 // 블록생성 메시지 전송 후 10초
	LeagueSelectValue  = 5  // 리그전송시 보내는 노드 수
	blockVoteWaiting   = 15 // 투표 대기
)

func (fu *FairTcp) sendLeague(otprnHash common.Hash) {
	t := time.NewTicker(sendLeagueCounter * time.Second)
	var percent float64 = LeaguePercent

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
			if num >= fu.JoinTotalNum(fu.manager.GetUsingOtprn(), percent) && num > 0 {
				fu.logger.Info("리그 전송", "리그 참여자 수", num)
				fu.sendTcpAll(otprnHash, transport.SendLeageNodeList, enodes)
				time.Sleep(makeJoinTxSig * time.Second)
				fu.manager.GetMakeJoinTxCh() <- struct{}{}
				return
			}

			if percent > reducePercent {
				// 최소 5%
				percent = percent - reducePercent
			}
		}
	}
}

func (fu *FairTcp) leagueControlle(otprnHash common.Hash) {
	fu.logger.Debug("leagueControlle Start", "otprnhash", otprnHash.String())
	defer fu.logger.Debug("leagueControlle kill ", "otprnhash", otprnHash.String())

	if fu.manager.GetUsingOtprn().HashOtprn() != otprnHash {
		return
	}

	for {
		select {
		case <-fu.manager.GetMakeJoinTxCh():
			fu.logger.Debug("조인 tx 생성", "otprnhash", otprnHash.String(), "blockNum", fu.manager.GetLastBlockNum().Uint64()+1)
			fu.sendTcpAll(otprnHash, transport.MakeJoinTx, fairtypes.BlockMakeMessage{otprnHash, fu.manager.GetLastBlockNum().Uint64() + 1})
			// 리그전송 4초  / 리그 내 블록 재생성 시작
			time.AfterFunc(time.Second, func() {

				fu.logger.Debug("블록 생성", "otprnhash", otprnHash.String(), "blockNum", fu.manager.GetLastBlockNum().Uint64()+1)
				fu.sendTcpAll(otprnHash, transport.MakeBlock, fairtypes.BlockMakeMessage{otprnHash, fu.manager.GetLastBlockNum().Uint64() + 1})

				time.AfterFunc(blockVote*time.Second, func() {
					fu.logger.Debug("블록 투표", "otprnhash", otprnHash.String(), "blockNum", fu.manager.GetLastBlockNum().Uint64()+1)
					fu.sendTcpAll(otprnHash, transport.WinningBlockVote, fairtypes.BlockMakeMessage{otprnHash, fu.manager.GetLastBlockNum().Uint64() + 1})
					go fu.sendFinalBlock(otprnHash)

				})

				//// peer list 전송후 14초 / 리그 내 블록 생성 후 10초
				//time.AfterFunc(finalBlockSig*time.Second, func() {
				//
				//})
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

	// 리그 보내는 메시지 코드 일때..
	if msgCode == transport.SendLeageNodeList {
		submitNode := SelectedNode(data)
		for index := range nodes {
			if nodes[index].Conn != nil {
				err := transport.Send(nodes[index].Conn, msgCode, submitNode[index])
				if err != nil {
					// 데이터 전송 오류 시 해당 연결 삭제
					fu.logger.Error("sendTcpAll", "msgcode", msgCode, "msg", err)
					nodes[index].Conn.Close()
					nodes[index].Conn = nil
				}
			}
		}
	} else {
		for index := range nodes {
			if nodes[index].Conn != nil {
				err := transport.Send(nodes[index].Conn, msgCode, data)
				if err != nil {
					// 데이터 전송 오류 시 해당 연결 삭제
					fu.logger.Error("sendTcpAll", "msgcode", msgCode, "msg", err)
					nodes[index].Conn.Close()
					nodes[index].Conn = nil
				}
			}
		}
	}
}

func GetNodeList(r *ring.Ring) []string {
	var res []string
	max := LeagueSelectValue

	if r.Len() < LeagueSelectValue {
		max = r.Len() - 1
	}

	for i := 0; i < max; i++ {
		r = r.Next()
		res = append(res, r.Value.(string))
	}
	return res
}

func SelectedNode(data interface{}) [][]string {
	var res [][]string
	sn, ok := data.([]string)
	if !ok {
		return nil
	}

	r := ring.New(len(sn))
	for i := 0; i < r.Len(); i++ {
		r.Value = sn[i]
		r = r.Next()
	}

	for i := range sn {
		r = r.Move(i)
		res = append(res, GetNodeList(r))
	}

	return res
}

func (fu *FairTcp) sendFinalBlock(otprnHash common.Hash) {
	fu.logger.Debug("sendFinalBlock Start", "otprnhash", otprnHash.String())
	defer fu.logger.Debug("sendFinalBlock kill ", "otprnhash", otprnHash.String())

	if fu.manager.GetUsingOtprn().HashOtprn() != otprnHash {
		return
	}

	votePool := fu.manager.GetVotePool()
	notify := make(chan *fairtypes.FinalBlock)

	time.AfterFunc(blockVoteWaiting*time.Second, func() {
		fu.logger.Debug("Get Final block start")
		go func() {
			t := time.NewTicker(time.Second)
			counter := 0
			for {
				select {
				case <-t.C:
					fb := fu.GetFinalBlock(otprnHash, votePool)
					if fb == nil {
						//10초 이후에 리그 교체 (투표 메시지 전달 후 10초 동안 투표가 없을경우)
						if counter == noVoteBolckLeagChn {
							notify <- nil
							return
						}
						counter++
						continue
					} else {
						notify <- fb
						return
					}
				}
			}
		}()
	})

	for {
		select {
		case n := <-notify:
			if n != nil {
				// DB에 블록 저장
				votePool.SnapShot <- n.Block
				votePool.DeleteCh <- pool.OtprnHash(otprnHash)

				// 파이널 블록 전송
				fu.sendTcpAll(otprnHash, transport.SendFinalBlock, n.GetTsFinalBlock())
				fu.logger.Debug("파이널 블록 전송", "blockNum", n.Block.NumberU64(), "miner", n.Block.Coinbase().String())

				time.Sleep(nextBlockMakeTerm * time.Second)
				fu.manager.GetManagerOtprnCh() <- struct{}{}
				return
			} else {
				fu.logger.Warn("파이널 블록 전송 시간 초과로 인한 리그 교체")
				fu.manager.GetStopLeagueCh() <- struct{}{}
				return
			}
		}
	}
}

func (fu *FairTcp) JoinTotalNum(otprn *types.Otprn, persent float64) uint64 {
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
		err := SignFairNode(fb.Block, voteBlocks[0].Voter, acc.ServerAcc, acc.KeyStore)
		if err != nil {
			fu.logger.Error("SignFairNode 서명에러", "error", err)
		}
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

		err := SignFairNode(fb.Block, pv.Voter, acc.ServerAcc, acc.KeyStore)
		if err != nil {
			fu.logger.Error("SignFairNode 서명에러", "error", err)
		}
	}

	return fb
}

func SignFairNode(block *types.Block, vBlock pool.VoteBlock, account accounts.Account, ks *keystore.KeyStore) error {
	_, err := ks.SignHash(account, block.Hash().Bytes())
	if err != nil {
		return err
	}

	// Fairnode block 서명 및 voter 추가
	//block.WithSealFairnode(sig)

	return nil
}
