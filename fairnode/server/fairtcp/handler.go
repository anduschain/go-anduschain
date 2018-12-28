package fairtcp

import (
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/fairnode/fairtypes"
	"github.com/anduschain/go-anduschain/fairnode/fairutil"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"github.com/anduschain/go-anduschain/fairnode/server/manager/pool"
	"github.com/anduschain/go-anduschain/fairnode/transport"
	"log"
)

func poolUpdate(leaguePool *pool.LeaguePool, otprn string, tsf fairtypes.TransferCheck) {
	leaguePool.UpdateCh <- pool.PoolIn{
		Hash: pool.StringToOtprn(otprn),
		Node: pool.Node{Enode: tsf.Enode, Coinbase: tsf.Coinbase, Conn: nil},
	}
}

func (ft *FairTcp) handelMsg(rw transport.MsgReadWriter) error {
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
		otprnHash := tsf.Otprn.HashOtprn().String()
		if ft.Db.CheckEnodeAndCoinbse(tsf.Enode, tsf.Coinbase.String()) {
			// TODO : andus >> 1. Enode가 맞는지 확인 ( 조회 되지 않으면 팅김 )
			// TODO : andus >> 2. 해당하는 Enode가 이전에 보낸 코인베이스와 일치하는지
			if fairutil.IsJoinOK(&tsf.Otprn, tsf.Coinbase) {
				// TODO : 채굴 리그 생성
				// TODO : 1. 채굴자 저장 ( key otprn num, Enode의 ID를 저장....)

				_, n, _ := ft.leaguePool.GetLeagueList(pool.StringToOtprn(otprnHash))
				if otprn.Mminer > n {
					log.Println("INFO : 참여 가능자 저장됨", tsf.Coinbase.String())
					ft.leaguePool.InsertCh <- pool.PoolIn{
						Hash: pool.StringToOtprn(otprnHash),
						Node: pool.Node{Enode: tsf.Enode, Coinbase: tsf.Coinbase, Conn: rw},
					}
				} else {
					// TODO : 참여 인원수 오버된 케이스
					log.Println("INFO : 참여 인원수 오버된 케이스", tsf.Enode)
					poolUpdate(ft.leaguePool, otprnHash, tsf)
				}
			} else {
				// TODO : andus >> 참여 대상자가 아니다
				log.Println("INFO : 참여 대상자가 아니다", tsf.Enode)
				poolUpdate(ft.leaguePool, otprnHash, tsf)
			}
		} else {
			// TODO : andus >> 리그 참여 정보가 다르다
			log.Println("INFO : 리그 참여 정보가 다르다", tsf.Enode)
			poolUpdate(ft.leaguePool, otprnHash, tsf)

		}
	case transport.SendBlockForVote:
		tsVote := fairtypes.TsVoteBlock{}
		if err := msg.Decode(&tsVote); err != nil {
			return err
		}

		voteBlock := tsVote.GetVoteBlock()
		block := voteBlock.Block
		otp := ft.manager.GetOtprn()
		lastNum := ft.manager.GetLastBlockNum()
		blockOtprnHash := common.BytesToHash(block.Extra())

		fmt.Println("블록 OTPRN : ", blockOtprnHash.String())
		fmt.Println("My OTPRN : ", blockOtprnHash.String())
		fmt.Println("받은 객체 OTPRN : ", voteBlock.OtprnHash.String())

		if otp.HashOtprn() == blockOtprnHash && lastNum+1 == block.NumberU64() {
			ft.manager.GetVotePool().InsertCh <- pool.Vote{
				Hash:     pool.StringToOtprn(voteBlock.OtprnHash.String()),
				Block:    block,
				Coinbase: voteBlock.Voter,
				//Receipts: voteBlock.Receipts,
			}
			fmt.Println("-----블록 투표 됨-----", voteBlock.Voter.String(), block.NumberU64())
		} else {
			fmt.Println("-----다른 OTPRN으로 투표 또는 숫자가 맞지 않아 거절됨-----", lastNum, otp.HashOtprn() == voteBlock.OtprnHash, block.Coinbase().String(), block.NumberU64())
		}
	default:
		return errors.New(fmt.Sprintf("알수 없는 메시지 코드 : %d", msg.Code))
	}

	return nil
}
