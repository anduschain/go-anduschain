package fairtcp

import (
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/fairutil"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/fairnode/server/manager/pool"
	"github.com/anduschain/go-anduschain/fairnode/transport"
	"log"
	"math/big"
)

func poolUpdate(leaguePool *pool.LeaguePool, otprnHash pool.OtprnHash, tsf fairtypes.TransferCheck) {
	leaguePool.UpdateCh <- pool.PoolIn{
		Hash: pool.OtprnHash(otprnHash),
		Node: pool.Node{Enode: tsf.Enode, Coinbase: tsf.Coinbase, Conn: nil},
	}
}

func (ft *FairTcp) handelMsg(rw transport.Transport, otprnHash common.Hash) error {
	msg, err := rw.ReadMsg()
	if err != nil {
		return err
	}
	defer msg.Discard()

	switch msg.Code {
	case transport.ReqLeagueJoinOK:
		tsf := fairtypes.TransferCheck{}
		if err := msg.Decode(&tsf); err != nil {
			return err
		}
		otprnHash := tsf.Otprn.HashOtprn()
		if ft.Db.CheckEnodeAndCoinbse(tsf.Enode, tsf.Coinbase.String()) {
			// TODO : andus >> 1. Enode가 맞는지 확인 ( 조회 되지 않으면 팅김 )
			// TODO : andus >> 2. 해당하는 Enode가 이전에 보낸 코인베이스와 일치하는지
			if fairutil.IsJoinOK(&tsf.Otprn, tsf.Coinbase) {
				// TODO : 채굴 리그 생성
				// TODO : 1. 채굴자 저장 ( key otprn num, Enode의 ID를 저장....)

				_, n, _ := ft.leaguePool.GetLeagueList(pool.OtprnHash(otprnHash))
				if otprn.Mminer > n {
					log.Println("INFO : 참여 가능자 저장됨", tsf.Coinbase.String())
					ft.leaguePool.InsertCh <- pool.PoolIn{
						Hash: pool.OtprnHash(otprnHash),
						Node: pool.Node{Enode: tsf.Enode, Coinbase: tsf.Coinbase, Conn: rw},
					}
				} else {
					// TODO : 참여 인원수 오버된 케이스
					log.Println("INFO : 참여 인원수 오버된 케이스", tsf.Enode)
					poolUpdate(ft.leaguePool, pool.OtprnHash(otprnHash), tsf)
				}
			} else {
				// TODO : andus >> 참여 대상자가 아니다
				log.Println("INFO : 참여 대상자가 아니다", tsf.Enode)
				poolUpdate(ft.leaguePool, pool.OtprnHash(otprnHash), tsf)
			}
		} else {
			// TODO : andus >> 리그 참여 정보가 다르다
			log.Println("INFO : 리그 참여 정보가 다르다", tsf.Enode)
			poolUpdate(ft.leaguePool, pool.OtprnHash(otprnHash), tsf)

		}
	case transport.SendBlockForVote:
		vote := fairtypes.Vote{}
		if err := msg.Decode(&vote); err != nil {
			return err
		}
		var currentBlockNum *big.Int
		if ft.manager.GetLastBlockNum() == nil {
			currentBlockNum = big.NewInt(1)
		} else {
			currentBlockNum = ft.manager.GetLastBlockNum().Add(ft.manager.GetLastBlockNum(), big.NewInt(1))
		}

		// block number check
		if vote.BlockNum.Cmp(currentBlockNum) != 0 {
			fmt.Println("--block number error---")
			break
		}

		// otprnhash check
		if ft.manager.GetUsingOtprn().HashOtprn() != vote.OtprnHash {
			fmt.Println("-otprnhash check error---")
			break
		}

		// sign check
		if !fairutil.ValidationSign(vote.HeaderHash.Bytes(), vote.Sig, vote.Voter) {
			fmt.Println("--sign check error---")
			break
		}

		ft.manager.GetVotePool().InsertCh <- pool.Vote{
			pool.OtprnHash(vote.OtprnHash), vote.HeaderHash, types.Voter{vote.Voter, vote.Sig},
		}

		fmt.Println("--블록 투표 됨--", vote.BlockNum.String(), vote.Voter.String())

	case transport.SendWinningBlock:
		tsblock := fairtypes.TsResWinningBlock{}
		if err := msg.Decode(&tsblock); err != nil {
			return err
		}
		ft.manager.GetVotePool().StoreBlockCh <- tsblock.GetResWinningBlock()
	default:
		return errors.New(fmt.Sprintf("알수 없는 메시지 코드 : %d", msg.Code))
	}

	return nil
}
