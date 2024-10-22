package client

import (
	"bytes"
	"errors"
	"github.com/anduschain/go-anduschain/core/types"
	proto "github.com/anduschain/go-anduschain/protos/common"
	"time"
)

// active miner heart beat
func (lc *Layer2Client) heartBeat() {
	t := time.NewTicker(HEART_BEAT_TERM * time.Minute)
	errCh := make(chan error)

	defer func() {
		lc.close()
		log.Warn("heart beat loop was dead")
	}()

	submit := func() error {
		lc.miner.Node.Head = lc.backend.BlockChain().CurrentHeader().Hash().String() // head change
		sign, err := lc.wallet.SignHash(lc.miner.Miner, lc.miner.Hash().Bytes())
		if err != nil {
			log.Error("heart beat sign node info", "msg", err)
			return err

		}
		lc.miner.Node.Sign = sign // heartbeat sign
		_, err = lc.rpc.HeartBeat(lc.ctx, &lc.miner.Node)
		if err != nil {
			log.Error("heart beat call", "msg", err)
			return err
		}
		log.Info("heart beat call", "message", lc.miner.Node.String())
		return nil
	}

	// init call
	if err := submit(); err != nil {
		return
	}

	go lc.requestOtprn(errCh) // otprn request

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

func (lc *Layer2Client) requestOtprn(errCh chan error) {
	t := time.NewTicker(REQ_OTPRN_TERM * time.Second)
	defer func() {
		errCh <- errors.New("request otprn error occurred")
		log.Warn("request otprn loop was dead")
	}()

	msg := proto.ReqOtprn{
		Enode:        lc.miner.Node.Enode,
		MinerAddress: lc.miner.Node.MinerAddress,
	}

	hash := rlpHash([]interface{}{
		msg.Enode,
		msg.MinerAddress,
	})

	sign, err := lc.wallet.SignHash(lc.miner.Miner, hash.Bytes())
	if err != nil {
		log.Error("heart beat sign node info", "msg", err)
		return
	}

	msg.Sign = sign // sign add

	reqOtprn := func() error {
		res, err := lc.rpc.RequestOtprn(lc.ctx, &msg)
		if res == nil || err != nil {
			log.Error("request otprn call", "msg", err)
			return err
		}

		switch res.Result {
		case proto.Status_SUCCESS:
			if bytes.Compare(res.Otprn, emptyByte) == 0 {
				log.Warn("Empty OTPRN")
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
				lc.mu.Lock()
				lc.otprn = otprn // otprn save
				lc.mu.Unlock()
				go lc.receiveOrdererTransactionLoop(*otprn)
				log.Info("otprn received and start orderer message loop", "hash", otprn.HashOtprn())
				return nil
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

func (lc *Layer2Client) receiveOrdererTransactionLoop(otprn types.Otprn) {
	defer log.Warn("receiveFairnodeStatusLoop was dead", "otprn", otprn.HashOtprn().String())
	msg := proto.Participate{
		Enode:        lc.miner.Node.Enode,
		MinerAddress: lc.miner.Node.MinerAddress,
		OtprnHash:    otprn.HashOtprn().Bytes(),
	}

	hash := rlpHash([]interface{}{
		msg.Enode,
		msg.MinerAddress,
		msg.OtprnHash,
	})

	sign, err := lc.wallet.SignHash(lc.miner.Miner, hash.Bytes())
	if err != nil {
		log.Error("Participate info signature", "msg", err)
		return
	}

	msg.Sign = sign

	stream, err := lc.rpc.ProcessController(lc.ctx, &msg)
	if err != nil {
		log.Error("ProcessController", "msg", err)
		return
	}

	defer stream.CloseSend()

	for {
		in, err := stream.Recv()
		if err != nil {
			log.Error("ProcessController stream receive", "msg", err)
			return
		}

		txLists := types.Transactions{}
		for idx, aTx := range in.Transactions {
			tx := types.Transaction{}
			err := tx.UnmarshalBinary(aTx.Transaction)
			if err != nil {
				log.Info("Transaction Unmarshal Error", "err", err)
			}
			sender, _ := tx.Sender(types.EIP155Signer{})

			log.Info("GOT TX", "idx", idx, "hash", tx.Hash(), "from", sender, "nonce", tx.Nonce())
			txLists = append(txLists, &tx)
		}
	}
}
