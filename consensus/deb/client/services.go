package client

import (
	"bytes"
	"errors"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/p2p/discover"
	proto "github.com/anduschain/go-anduschain/protos/common"
	"math/big"
	"time"
)

// active miner heart beat
func (dc *DebClient) heartBeat() {
	t := time.NewTicker(HEART_BEAT_TERM * time.Minute)
	errCh := make(chan error)

	defer func() {
		dc.close()
		log.Warn("heart beat loop was dead")
	}()

	sign, err := dc.wallet.SignHash(dc.miner.Miner, dc.miner.Hash().Bytes())
	if err != nil {
		log.Error("heart beat sign node info", "msg", err)
		return
	}

	dc.miner.Node.Sign = sign // heartbeat sign

	submit := func() error {
		_, err := dc.rpc.HeartBeat(dc.ctx, &dc.miner.Node)
		if err != nil {
			log.Error("heart beat call", "msg", err)
			return err
		}
		log.Info("heart beat call", "message", dc.miner.Node.String())
		return nil
	}

	// init call
	if err := submit(); err != nil {
		return
	}

	go dc.requestOtprn(errCh) // otprn request

	for {
		select {
		case <-t.C:
			if err := submit(); err != nil {
				return
			}
		case err := <-errCh:
			log.Error("heartBeat loop was dead", "msg", err)
			return
		}
	}
}

func (dc *DebClient) requestOtprn(errCh chan error) {
	t := time.NewTicker(REQ_OTPRN_TERM * time.Second)
	defer func() {
		errCh <- errors.New("request otprn error occurred")
		log.Warn("request otprn loop was dead")
	}()

	msg := proto.ReqOtprn{
		Enode:        dc.miner.Node.Enode,
		MinerAddress: dc.miner.Node.MinerAddress,
	}

	hash := rlpHash([]interface{}{
		msg.Enode,
		msg.MinerAddress,
	})

	sign, err := dc.wallet.SignHash(dc.miner.Miner, hash.Bytes())
	if err != nil {
		log.Error("heart beat sign node info", "msg", err)
		return
	}

	msg.Sign = sign // sign add

	reqOtprn := func() error {
		res, err := dc.rpc.RequestOtprn(dc.ctx, &msg)
		if res == nil || err != nil {
			log.Error("request otprn call", "msg", err)
			return err
		}

		switch res.Result {
		case proto.Status_SUCCESS:
			if bytes.Compare(res.Otprn, emptyByte) == 0 {
				log.Warn("do not participate in this league")
				return nil
			} else {
				otprn, err := types.DecodeOtprn(res.Otprn)
				if err != nil {
					log.Error("decode otprn call", "msg", err)
					return err
				}

				for _, o := range dc.otprn {
					if o.HashOtprn() == otprn.HashOtprn() {
						return nil
					}
				}

				dc.otprn = append(dc.otprn, otprn) // otprn save
				go dc.receiveFairnodeStatusLoop(*otprn)
			}
		case proto.Status_FAIL:
			log.Warn("otprn got nil")
			return nil
		}

		return nil
	}

	// init call
	if err := reqOtprn(); err != nil {
		return
	}

	for {
		select {
		case <-t.C:
			if err := reqOtprn(); err != nil {
				return
			}
		}
	}
}

func (dc *DebClient) receiveFairnodeStatusLoop(otprn types.Otprn) {
	defer log.Warn("receiveFairnodeStatusLoop was dead", "otprn", otprn.HashOtprn().String())

	msg := proto.Participate{
		Enode:        dc.miner.Node.Enode,
		MinerAddress: dc.miner.Node.MinerAddress,
		OtprnHash:    otprn.HashOtprn().Bytes(),
	}

	hash := rlpHash([]interface{}{
		msg.Enode,
		msg.MinerAddress,
		msg.OtprnHash,
	})

	sign, err := dc.wallet.SignHash(dc.miner.Miner, hash.Bytes())
	if err != nil {
		log.Error("Participate info signature", "msg", err)
		return
	}

	msg.Sign = sign

	stream, err := dc.rpc.ProcessController(dc.ctx, &msg)
	if err != nil {
		log.Error("ProcessController", "msg", err)
		return
	}

	defer stream.CloseSend()

	var (
		isReqLeague bool
		isJoinTx    bool
		isVote      bool
	)

	for {
		in, err := stream.Recv()
		if err != nil {
			log.Error("ProcessController stream receive", "msg", err)
			return
		}

		hash := rlpHash([]interface{}{
			in.Code,
		})

		if err := ValidationSignHash(in.GetSign(), hash, dc.FnAddress()); err != nil {
			log.Error("VerifySignature", "msg", err)
			return
		}

		switch in.GetCode() {
		case proto.ProcessStatus_MAKE_LEAGUE:
			if !isReqLeague {
				enodes := dc.requestLeague(otprn) // 해당 리그에 해당되는 노드 리스트
				for _, enode := range enodes {
					dc.backend.Server().AddPeer(discover.MustParseNode(enode))
					log.Info("make league status", "addPeer", enodes)
				}
				isReqLeague = true
			}
		case proto.ProcessStatus_MAKE_JOIN_TX:
			if isJoinTx {
				continue
			}

			fnBlockNum := new(big.Int)
			fnBlockNum.SetBytes(in.GetCurrentBlockNum())
			current := dc.backend.BlockChain().CurrentHeader().Number
			if current.Cmp(fnBlockNum) == 0 {
				// make join transaction
				state := dc.backend.TxPool().State()
				coinbase := dc.miner.Miner.Address

				nonce := state.GetNonce(coinbase)
				jnonce := state.GetJoinNonce(coinbase)

				bOtrpn, err := otprn.EncodeOtprn()
				if err != nil {
					log.Error("otprn encode err", "msg", err)
					return
				}

				sTx, err := dc.wallet.SignTx(dc.miner.Miner, types.NewJoinTransaction(nonce, jnonce, bOtrpn), dc.config.ChainID)
				if err != nil {
					log.Error("signature join transaction", "msg", err)
					return
				}

				if err := dc.backend.TxPool().AddLocal(sTx); err != nil {
					log.Error("join transaction add local", "msg", err)
					return
				}

				isJoinTx = true
				log.Info("made join transaction", "hash", sTx.Hash())
			} else {
				log.Warn("fail made join transaction", "fnBlockNum", fnBlockNum.String(), "current", current.String())
			}
		case proto.ProcessStatus_MAKE_BLOCK:
			dc.statusFeed.Send(types.FairnodeStatusEvent{Status: types.MAKE_BLOCK, Payload: otprn})
		case proto.ProcessStatus_VOTE_START:
			if isVote {
				continue
			}
			voteCh := make(chan types.NewLeagueBlockEvent)
			dc.statusFeed.Send(types.FairnodeStatusEvent{Status: types.VOTE_START, Payload: voteCh})
			select {
			case ev := <-voteCh:
				dc.vote(ev)
				isVote = true
			}
		case proto.ProcessStatus_VOTE_COMPLETE:
			dc.statusFeed.Send(types.FairnodeStatusEvent{Status: types.VOTE_COMPLETE})

		default:
			log.Info("receiveFairnodeStatusLoop", "stream", in.GetCode().String()) // TODO(hakuna) : change level -> trace
		}
	}
}

func (dc *DebClient) requestLeague(otprn types.Otprn) []string {
	msg := proto.ReqLeague{
		Enode:        dc.miner.Node.Enode,
		MinerAddress: dc.miner.Node.MinerAddress,
		OtprnHash:    otprn.HashOtprn().Bytes(),
	}

	hash := rlpHash([]interface{}{
		msg.Enode,
		msg.MinerAddress,
		msg.OtprnHash,
	})

	sign, err := dc.wallet.SignHash(dc.miner.Miner, hash.Bytes())
	if err != nil {
		log.Error("requestLeague info signature", "msg", err)
		return nil
	}

	msg.Sign = sign

	res, err := dc.rpc.RequestLeague(dc.ctx, &msg)
	if err != nil {
		log.Error("request league", "msg", err)
		return nil
	}

	if res.GetResult() == proto.Status_SUCCESS {
		log.Info("request league received", "count", len(res.GetEnodes()))
		return res.GetEnodes()
	} else {
		return nil
	}
}

func (dc *DebClient) vote(ev types.NewLeagueBlockEvent) {
	block := ev.Block
	if block == nil {
		log.Error("voting block is nil")
		return
	}

	msg := proto.Vote{
		Header:       block.Header().Byte(),
		VoterAddress: ev.Address.String(),
	}

	hash := rlpHash([]interface{}{
		msg.Header,
		msg.VoterAddress,
	})

	sign, err := dc.wallet.SignHash(dc.miner.Miner, hash.Bytes())
	if err != nil {
		log.Error("voting info signature", "msg", err)
		return
	}

	msg.VoterSign = sign // add voter's signature

	_, err = dc.rpc.Vote(dc.ctx, &msg)
	if err != nil {
		log.Error("voting request", "msg", err)
		return
	}

	log.Info("vote success", "hash", block.Hash())
}
