package client

import (
	"bytes"
	"errors"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/verify"
	"github.com/anduschain/go-anduschain/orderer"
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

	go lc.receiveOrdererTransactionLoop(errCh) // otprn request

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

func (lc *Layer2Client) requestOtprn() error {
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
		log.Error("requestOtprn", "msg", err)
		return err
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
				return errors.New("Empty OTPRN")
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
				log.Info("otprn received", "hash", otprn.HashOtprn())
				return nil
			}
		case proto.Status_FAIL:
			log.Debug("otprn got nil")
			return errors.New("otprn got nil")
		}
		return errors.New("Unknown Error")
	}

	// init call
	return reqOtprn()
}

func (lc *Layer2Client) receiveOrdererTransactionLoop(errCh chan error) {
	defer func() {
		errCh <- errors.New("receiveOrdererTransactionLoop was dead")
		log.Warn("receiveOrdererTransactionLoop was dead")
	}()

	if err := lc.requestOtprn(); err != nil {
		errCh <- err
	}

	msg := proto.Participate{
		Enode:        lc.miner.Node.Enode,
		MinerAddress: lc.miner.Node.MinerAddress,
		OtprnHash:    lc.otprn.HashOtprn().Bytes(),
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
		rHash := common.Hash{}
		for idx, aTx := range in.Transactions {
			tx := types.Transaction{}
			err := tx.UnmarshalBinary(aTx.Transaction)
			if err != nil {
				log.Info("Transaction Unmarshal Error", "err", err)
			}
			hash := tx.Hash()
			thash, err := orderer.XorHashes(rHash.String(), hash.String())
			if err == nil {
				rHash = common.HexToHash(thash)
			}
			sender, _ := tx.Sender(types.EIP155Signer{})

			log.Info("GOT TX", "idx", idx, "hash", tx.Hash(), "from", sender, "nonce", tx.Nonce())
			txLists = append(txLists, &tx)
		}
		// TXLIST 검증
		err = verify.ValidationSignHash(in.Sign, rHash, lc.otprn.FnAddr)
		if err != nil {
			log.Error("ProcessController Orderer sign Velidation Error", "msg", err)
			return
		}
		// 이 리스트를 이용하여 블록생성.. 채굴과정으로 넘어가야 함
	}
}
