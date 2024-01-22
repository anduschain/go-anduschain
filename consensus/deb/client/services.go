package client

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/common/math"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/verify"
	"github.com/anduschain/go-anduschain/p2p"
	"github.com/anduschain/go-anduschain/p2p/discover"
	"github.com/anduschain/go-anduschain/params"
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

	submit := func() error {
		dc.miner.Node.Head = dc.backend.BlockChain().CurrentHeader().Hash().String() // head change
		sign, err := dc.wallet.SignHash(dc.miner.Miner, dc.miner.Hash().Bytes())
		if err != nil {
			log.Error("heart beat sign node info", "msg", err)
			return err

		}
		dc.miner.Node.Sign = sign // heartbeat sign
		_, err = dc.rpc.HeartBeat(dc.ctx, &dc.miner.Node)
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

				if err := otprn.ValidateSignature(); err != nil {
					return err
				}
				dc.mu.Lock()
				if _, ok := dc.otprn[otprn.HashOtprn()]; ok {
					dc.mu.Unlock()
					log.Debug("already, have been had otprn", "msg", err)
					return nil
				} else {
					dc.otprn[otprn.HashOtprn()] = otprn // otprn save
					dc.mu.Unlock()
					go dc.receiveFairnodeStatusLoop(*otprn)
					log.Info("otprn received and start fairnode status loop", "hash", otprn.HashOtprn())
					return nil
				}

			}
		case proto.Status_FAIL:
			log.Debug("otprn got nil")
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

func (dc *DebClient) disconnectNonStatic() {
	peers := dc.backend.Server().Peers()
	log.Info("League End: disconnect non static peer", "peers", len(peers))
	if len(peers) > dc.backend.Server().MaxPeers/3*2 {
		for idx, peer := range peers {
			if !peer.Info().Network.Static {
				if idx%2 == 0 {
					log.Info("Disconnect peer", "id", peer.ID())
					peer.Disconnect(p2p.DiscQuitting)
				}
			}
		}
	}
}

func (dc *DebClient) receiveFairnodeStatusLoop(otprn types.Otprn) {
	defer log.Warn("receiveFairnodeStatusLoop was dead", "otprn", otprn.HashOtprn().String())
	defer dc.disconnectNonStatic()

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

	var stCode proto.ProcessStatus

	for {
		in, err := stream.Recv()
		if err != nil {
			log.Error("ProcessController stream receive", "msg", err)
			return
		}
		hash := rlpHash([]interface{}{
			in.Code,
			in.CurrentBlockNum,
		})
		if err := verify.ValidationSignHash(in.GetSign(), hash, dc.FnAddress()); err != nil {
			log.Error("VerifySignature", "msg", err)
			return
		}
		log.Debug("Receive fairnode signal", "hash", otprn.HashOtprn(), "stream", in.GetCode().String())
		if stCode != in.GetCode() {
			stCode = in.GetCode()
		} else {
			continue
		}
		switch stCode {
		case proto.ProcessStatus_MAKE_LEAGUE:
			enodes := dc.requestLeague(otprn) // 해당 리그에 해당되는 노드 리스트
			// TODO CSW: static node -> dynamic node
			dc.backend.Server().DeleteStaticPeers(dc.staticNodes)
			// TODO CSW: convert ip-address using local-ips.json
			for i, enode := range enodes {
				eNode := enode
				id, host, port := common.SplitEnode(enode)
				if host != "" {
					val := dc.localIps[host]
					if val != "" {
						eNode = id + "@" + val + ":" + port
					}
				}
				dc.backend.Server().AddPeer(discover.MustParseNode(eNode))
				log.Info("make league status", "addPeer", enodes[i], "realPeer", eNode)
			}
		case proto.ProcessStatus_MAKE_JOIN_TX:
			fnBlockNum := new(big.Int)
			fnBlockNum.SetBytes(in.GetCurrentBlockNum())
			current := dc.backend.BlockChain().CurrentHeader().Number
			if current.Cmp(fnBlockNum) == 0 {
				// make join transaction
				state := dc.backend.TxPool().State()
				coinbase := dc.miner.Miner.Address
				// balance check
				epoch := new(big.Float).SetUint64(otprn.Data.Epoch)                  // block count which league will be made
				fee, _ := new(big.Float).SetString(otprn.Data.Price.JoinTxPrice)     // join transaction fee
				price := new(big.Float).Mul(big.NewFloat(params.Daon), fee)          // join transaction price ( fee * 10e18) - unit : daon
				limitBalance := math.FloatToBigInt(new(big.Float).Mul(price, epoch)) // minimum balance for participate in league.
				balance := state.GetBalance(coinbase)                                // current balance
				if balance.Cmp(limitBalance) < 0 {
					log.Error("not enough balance", "limit", fmt.Sprintf("%s Wie", limitBalance.String()), "balance", fmt.Sprintf("%s Wie", balance.String()))
					return
				}
				nonce := state.GetNonce(coinbase)
				jnonce := state.GetJoinNonce(coinbase)
				bOtrpn, err := otprn.EncodeOtprn()
				if err != nil {
					log.Error("otprn encode err", "msg", err)
					return
				}

				sTx, err := dc.wallet.SignTx(dc.miner.Miner, types.NewJoinTransaction(nonce, jnonce, bOtrpn, dc.miner.Miner.Address), dc.config.ChainID)
				if err != nil {
					log.Error("signature join transaction", "msg", err)
					return
				}
				if err := dc.backend.TxPool().AddLocal(sTx); err != nil {
					log.Error("join transaction add local", "msg", err)
					return
				}
				log.Info("made join transaction", "hash", sTx.Hash())
			} else {
				log.Error("fail made join transaction", "fnBlockNum", fnBlockNum.String(), "current", current.String())
				return
			}
		case proto.ProcessStatus_MAKE_BLOCK:
			dc.statusFeed.Send(types.FairnodeStatusEvent{Status: types.MAKE_BLOCK, Payload: otprn})
		case proto.ProcessStatus_LEAGUE_BROADCASTING:
			dc.statusFeed.Send(types.FairnodeStatusEvent{Status: types.LEAGUE_BROADCASTING, Payload: nil})
		case proto.ProcessStatus_VOTE_START:
			voteCh := make(chan types.NewLeagueBlockEvent)
			dc.statusFeed.Send(types.FairnodeStatusEvent{Status: types.VOTE_START, Payload: voteCh})
			select {
			case ev := <-voteCh:
				dc.vote(ev)
			}
		case proto.ProcessStatus_VOTE_COMPLETE:
			// 투표결과 요청 후, voter 확인 후, 블록에 넣기
			voters := dc.requestVoteResult(otprn)
			if voters == nil {
				continue
			}
			submitBlockCh := make(chan *types.Block)
			dc.statusFeed.Send(types.FairnodeStatusEvent{Status: types.VOTE_COMPLETE, Payload: []interface{}{voters, submitBlockCh}})
			select {
			case block := <-submitBlockCh:
				go dc.reqSealConfirm(otprn, *block)
			}
		case proto.ProcessStatus_FINALIZE:
			// make block routine start
			dc.statusFeed.Send(types.FairnodeStatusEvent{Status: types.FINALIZE, Payload: nil})
		case proto.ProcessStatus_REJECT:
			delete(dc.otprn, otprn.HashOtprn()) // delete otprn
			log.Warn("Fairnode status loop", "msg", "otprn deleted")
		default:
			log.Info("Receive Fairnode Status Loop", "stream", in.GetCode().String()) // TODO(hakuna) : change level -> trace
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
		log.Error("Voting block is nil")
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
		log.Error("Voting info signature", "msg", err)
		return
	}

	msg.VoterSign = sign // add voter's signature

	_, err = dc.rpc.Vote(dc.ctx, &msg)
	if err != nil {
		log.Error("Voting request", "msg", err)
		return
	}

	log.Info("Vote Success", "hash", block.Hash())
}

func (dc *DebClient) requestVoteResult(otprn types.Otprn) types.Voters {
	msg := proto.ReqVoteResult{
		OtprnHash: otprn.HashOtprn().Bytes(),
		Address:   dc.miner.Node.MinerAddress,
	}

	hash := rlpHash([]interface{}{
		msg.OtprnHash,
		msg.Address,
	})

	sign, err := dc.wallet.SignHash(dc.miner.Miner, hash.Bytes())
	if err != nil {
		log.Error("voting info signature", "msg", err)
		return nil
	}

	msg.Sign = sign

	res, err := dc.rpc.RequestVoteResult(dc.ctx, &msg)
	if err != nil {
		log.Error("Request Vote Result", "msg", err)
		return nil
	}

	hash = rlpHash([]interface{}{
		res.GetResult(),
		res.GetBlockHash(),
		res.GetVoters(),
	})

	if err := verify.ValidationSignHash(res.GetSign(), hash, dc.FnAddress()); err != nil {
		log.Error("VerifySignature", "msg", err)
		return nil
	}

	if res.Result == proto.Status_SUCCESS {
		var voters []*types.Voter
		for _, vote := range res.GetVoters() {
			hash := rlpHash([]interface{}{
				vote.Header,
				vote.VoterAddress,
			})

			err := verify.ValidationSignHash(vote.GetVoterSign(), hash, common.HexToAddress(vote.VoterAddress))
			if err != nil {
				log.Error("VerifySignature", "msg", err)
				return nil
			}

			voters = append(voters, &types.Voter{
				Header:   vote.Header,
				Voter:    common.HexToAddress(vote.VoterAddress),
				VoteSign: vote.VoterSign,
			})
		}
		return types.Voters(voters)
	} else {
		return nil
	}
}

func (dc *DebClient) reqSealConfirm(otprn types.Otprn, block types.Block) {
	defer log.Warn("reqSealConfirm was dead", "otprn", otprn.HashOtprn().String())
	msg := proto.ReqConfirmSeal{
		OtprnHash: otprn.HashOtprn().Bytes(),
		Address:   dc.miner.Node.MinerAddress,
		BlockHash: block.Hash().Bytes(),
		VoteHash:  block.VoterHash().Bytes(),
	}
	hash := rlpHash([]interface{}{
		msg.GetOtprnHash(),
		msg.GetBlockHash(),
		msg.GetVoteHash(),
		msg.GetAddress(),
	})
	sign, err := dc.wallet.SignHash(dc.miner.Miner, hash.Bytes())
	if err != nil {
		log.Error("request SealConfirm info signature", "msg", err)
		return
	}
	msg.Sign = sign
	stream, err := dc.rpc.SealConfirm(dc.ctx, &msg)
	if err != nil {
		log.Error("request SealConfirm", "msg", err)
		return
	}
	defer stream.CloseSend()
	var stCode proto.ProcessStatus
	for {
		in, err := stream.Recv()
		if err != nil {
			log.Error("Request SealConfirm stream receive", "msg", err)
			return
		}
		hash := rlpHash([]interface{}{
			in.Code,
		})
		if err := verify.ValidationSignHash(in.GetSign(), hash, dc.FnAddress()); err != nil {
			log.Error("VerifySignature", "msg", err)
			return
		}
		log.Debug("Request SealConfirm Status", "hash", otprn.HashOtprn(), "stream", in.GetCode().String())
		if stCode != in.GetCode() {
			stCode = in.GetCode()
		} else {
			continue
		}
		switch stCode {
		case proto.ProcessStatus_SEND_BLOCK:
			// submitting to fairnode
			go dc.sendBlock(block)
		case proto.ProcessStatus_REQ_FAIRNODE_SIGN:
			fnSign := dc.requestFairnodeSign(otprn, block)
			if fnSign == nil {
				time.Sleep(200 * time.Millisecond)
				continue
			} else {
				dc.statusFeed.Send(types.FairnodeStatusEvent{Status: types.REQ_FAIRNODE_SIGN, Payload: fnSign})
				return
			}
		default:
			log.Info("Request SealConfirm loop", "stream", in.GetCode().String()) // TODO(hakuna) : change level -> trace
		}
	}
}

func (dc *DebClient) sendBlock(block types.Block) {
	defer log.Info("Send Block To Fairnode", "hash", block.Hash())
	var buf bytes.Buffer
	err := block.EncodeRLP(&buf)
	if err != nil {
		log.Error("Send block EncodeRLP", "msg", err)
		return
	}
	msg := proto.ReqBlock{
		Block:   buf.Bytes(),
		Address: dc.miner.Node.MinerAddress,
	}
	hash := rlpHash([]interface{}{
		msg.GetBlock(),
		msg.GetAddress(),
	})
	sign, err := dc.wallet.SignHash(dc.miner.Miner, hash.Bytes())
	if err != nil {
		log.Error("Send block signature", "msg", err)
		return
	}
	msg.Sign = sign
	_, err = dc.rpc.SendBlock(dc.ctx, &msg)
	if err != nil {
		log.Error("Send block, rpc send", "msg", err)
		return
	}
}

func (dc *DebClient) requestFairnodeSign(otprn types.Otprn, block types.Block) []byte {
	msg := proto.ReqFairnodeSign{
		OtprnHash: otprn.HashOtprn().Bytes(),
		Address:   dc.miner.Node.MinerAddress,
		BlockHash: block.Hash().Bytes(),
		VoteHash:  block.VoterHash().Bytes(),
	}
	hash := rlpHash([]interface{}{
		msg.GetOtprnHash(),
		msg.GetBlockHash(),
		msg.GetVoteHash(),
		msg.GetAddress(),
	})
	sign, err := dc.wallet.SignHash(dc.miner.Miner, hash.Bytes())
	if err != nil {
		log.Error("Request Fairnode signature info signature", "msg", err)
		return nil
	}
	msg.Sign = sign
	res, err := dc.rpc.RequestFairnodeSign(dc.ctx, &msg)
	if err != nil {
		log.Error("Request Fairnode signature", "msg", err)
		return nil
	}
	hash = rlpHash([]interface{}{
		res.GetSignature(),
	})
	if err := verify.ValidationSignHash(res.GetSign(), hash, dc.FnAddress()); err != nil {
		log.Error("VerifySignature", "msg", err)
		return nil
	}

	return res.GetSignature()
}
