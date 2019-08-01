package fairnode

import (
	"bytes"
	"context"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairdb"
	"github.com/anduschain/go-anduschain/fairnode/verify"
	"github.com/anduschain/go-anduschain/log"
	proto "github.com/anduschain/go-anduschain/protos/common"
	"github.com/anduschain/go-anduschain/protos/fairnode"
	"github.com/anduschain/go-anduschain/rlp"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc/peer"
	"math/big"
	"strings"
	"time"
)

var (
	emptyByte []byte
)

type fnNode interface {
	SignHash(hash []byte) ([]byte, error)
	Database() fairdb.FairnodeDB
	LeagueSet() map[common.Hash]*league
}

func errorEmpty(key string) error {
	return errors.New(fmt.Sprintf("%s value is empty", key))
}

// fairnode rpc method implemented
type rpcServer struct {
	fn      fnNode
	db      fairdb.FairnodeDB
	leagues map[common.Hash]*league
}

func newServer(fn fnNode) *rpcServer {
	return &rpcServer{
		fn:      fn,
		db:      fn.Database(),
		leagues: fn.LeagueSet(),
	}
}

// Heart Beat : notify to fairnode, I'm alive.
func (rs *rpcServer) HeartBeat(ctx context.Context, nodeInfo *proto.HeartBeat) (*empty.Empty, error) {
	p, _ := peer.FromContext(ctx)
	ip, err := ParseIP(p.Addr.String())
	if err != nil {
		logger.Error("HeartBeat ParseIP", "msg", err)
		return nil, err
	}

	if nodeInfo.GetEnode() == "" {
		return nil, errorEmpty("enode")
	}

	if nodeInfo.GetChainID() == "" {
		return nil, errorEmpty("chainID")
	}

	if nodeInfo.GetMinerAddress() == "" {
		return nil, errorEmpty("miner's address")
	}

	if nodeInfo.GetPort() < 1025 {
		return nil, errors.New("invalid port number")
	}

	if nodeInfo.GetNodeVersion() == "" {
		return nil, errorEmpty("node version")
	}

	if nodeInfo.GetHead() == "" {
		return nil, errorEmpty("head")
	}

	if bytes.Compare(nodeInfo.GetSign(), emptyByte) == 0 {
		return nil, errorEmpty("sign")
	}

	hash := rlpHash([]interface{}{
		nodeInfo.GetEnode(),
		nodeInfo.GetNodeVersion(),
		nodeInfo.GetChainID(),
		nodeInfo.GetMinerAddress(),
		nodeInfo.GetPort(),
		nodeInfo.GetHead(),
	})

	err = verify.ValidationSignHash(nodeInfo.GetSign(), hash, common.HexToAddress(nodeInfo.GetMinerAddress()))
	if err != nil {
		return nil, errors.New(fmt.Sprintf("sign validation failed msg=%s", err.Error()))
	}

	if cBlock := rs.db.CurrentBlock(); cBlock != nil {
		if cBlock.Hash().String() != nodeInfo.GetHead() {
			if block := rs.db.GetBlock(common.HexToHash(nodeInfo.GetHead())); block != nil {
				if cBlock.Number().Uint64()-block.Number().Uint64() != 1 {
					return nil, errors.New("head is mismatch")
				}
			} else {
				return nil, errors.New("head is mismatch")
			}
		}
	}

	rs.db.SaveActiveNode(types.HeartBeat{
		Enode:        nodeInfo.GetEnode(),
		NodeVersion:  nodeInfo.GetNodeVersion(),
		ChainID:      nodeInfo.GetChainID(),
		MinerAddress: nodeInfo.GetMinerAddress(),
		Host:         ip,
		Port:         nodeInfo.GetPort(),
		Head:         common.HexToHash(nodeInfo.GetHead()),
		Time:         big.NewInt(time.Now().Unix()),
	})

	defer logger.Info("HeartBeat received", "enode", reduceStr(nodeInfo.GetEnode()))
	return &empty.Empty{}, nil
}

func (rs *rpcServer) RequestOtprn(ctx context.Context, nodeInfo *proto.ReqOtprn) (*proto.ResOtprn, error) {
	if nodeInfo.GetEnode() == "" {
		return nil, errorEmpty("enode")
	}

	if nodeInfo.GetMinerAddress() == "" {
		return nil, errorEmpty("miner's address")
	}

	if bytes.Compare(nodeInfo.GetSign(), emptyByte) == 0 {
		return nil, errorEmpty("sign")
	}

	hash := rlpHash([]interface{}{
		nodeInfo.GetEnode(),
		nodeInfo.GetMinerAddress(),
	})

	addr := common.HexToAddress(nodeInfo.GetMinerAddress())

	err := verify.ValidationSignHash(nodeInfo.GetSign(), hash, addr)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("sign validation failed msg=%s", err.Error()))
	}

	// heart beat 리스트에 있는지 확인
	IsExist := false
	for _, node := range rs.db.GetActiveNode() {
		if strings.Compare(node.Enode, nodeInfo.GetEnode()) == 0 {
			IsExist = true
			break
		}
	}

	if !IsExist {
		return nil, errors.New(fmt.Sprintf("does not exist in active node list addr = %s", reduceStr(nodeInfo.GetEnode())))
	}

	if otprn := rs.db.CurrentOtprn(); otprn != nil {
		err := otprn.ValidateSignature()
		if err != nil {
			return nil, errors.New(fmt.Sprintf("otprn validation failed msg=%s", err.Error()))
		}

		// 참가 대상 확인
		if verify.IsJoinOK(otprn, addr) {
			// OTPRN 전송
			bOtprn, err := otprn.EncodeOtprn()
			if err != nil {
				return nil, errors.New(fmt.Sprintf("otprn EncodeOtprn failed msg=%s", err.Error()))
			}

			rs.db.SaveLeague(otprn.HashOtprn(), nodeInfo.GetEnode()) // 리그 리스트에 저장

			logger.Info("otprn submitted", "otrpn", reduceStr(otprn.HashOtprn().String()), "enode", reduceStr(nodeInfo.GetEnode()))
			return &proto.ResOtprn{
				Result: proto.Status_SUCCESS,
				Otprn:  bOtprn,
			}, nil
		} else {
			logger.Warn("otprn not submitted", "msg", "Not eligible", "enode", reduceStr(nodeInfo.GetEnode()))
			return &proto.ResOtprn{
				Result: proto.Status_SUCCESS,
			}, nil
		}
	} else {
		return &proto.ResOtprn{
			Result: proto.Status_FAIL,
		}, nil
	}
}

func (rs *rpcServer) RequestLeague(ctx context.Context, nodeInfo *proto.ReqLeague) (*proto.ResLeague, error) {
	if nodeInfo.GetEnode() == "" {
		return nil, errorEmpty("enode")
	}

	if nodeInfo.GetMinerAddress() == "" {
		return nil, errorEmpty("miner's address")
	}

	if bytes.Compare(nodeInfo.GetOtprnHash(), emptyByte) == 0 {
		return nil, errorEmpty("otprn hash")
	}

	if bytes.Compare(nodeInfo.GetSign(), emptyByte) == 0 {
		return nil, errorEmpty("sign")
	}

	hash := rlpHash([]interface{}{
		nodeInfo.GetEnode(),
		nodeInfo.GetMinerAddress(),
		nodeInfo.GetOtprnHash(),
	})

	addr := common.HexToAddress(nodeInfo.GetMinerAddress())

	err := verify.ValidationSignHash(nodeInfo.GetSign(), hash, addr)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("sign validation failed msg=%s", err.Error()))
	}

	otprn := common.BytesToHash(nodeInfo.GetOtprnHash())
	nodes := rs.db.GetLeagueList(otprn)
	enodes := SelectedNode(nodeInfo.GetEnode(), nodes, 5) // 5 enode url per node
	if enodes == nil {
		return nil, errors.New("empty enode list")
	}

	m := proto.ResLeague{
		Result: proto.Status_SUCCESS,
		Enodes: enodes,
	}

	hash = rlpHash([]interface{}{
		m.GetResult(),
		m.GetEnodes(),
	})

	sign, err := rs.fn.SignHash(hash.Bytes())
	if err != nil {
		logger.Error("RequestLeague signature message", "msg", err)
		return nil, err
	}

	m.Sign = sign // add sign
	return &m, nil
}

func (rs *rpcServer) ProcessController(nodeInfo *proto.Participate, stream fairnode.FairnodeService_ProcessControllerServer) error {
	if nodeInfo.GetEnode() == "" {
		return errorEmpty("enode")
	}

	if nodeInfo.GetMinerAddress() == "" {
		return errorEmpty("miner's address")
	}

	if bytes.Compare(nodeInfo.GetOtprnHash(), emptyByte) == 0 {
		return errorEmpty("otprn hash")
	}

	if bytes.Compare(nodeInfo.GetSign(), emptyByte) == 0 {
		return errorEmpty("sign")
	}

	hash := rlpHash([]interface{}{
		nodeInfo.GetEnode(),
		nodeInfo.GetMinerAddress(),
		nodeInfo.GetOtprnHash(),
	})

	addr := common.HexToAddress(nodeInfo.GetMinerAddress())
	err := verify.ValidationSignHash(nodeInfo.GetSign(), hash, addr)
	if err != nil {
		return errors.New(fmt.Sprintf("sign validation failed msg=%s", err.Error()))
	}

	otprnHash := common.BytesToHash(nodeInfo.GetOtprnHash())
	var clg *league // current league
	if league, ok := rs.leagues[otprnHash]; ok {
		clg = league
	} else {
		return errors.New(fmt.Sprintf("this otprn is not matched in league hash=%s", nodeInfo.GetOtprnHash()))
	}

	makeMsg := func(l *league) *proto.ProcessMessage {
		var msg proto.ProcessMessage
		switch l.Status {
		case types.PENDING:
			msg.Code = proto.ProcessStatus_WAIT
		case types.MAKE_LEAGUE:
			msg.Code = proto.ProcessStatus_MAKE_LEAGUE
		case types.MAKE_JOIN_TX:
			msg.Code = proto.ProcessStatus_MAKE_JOIN_TX
		case types.MAKE_BLOCK:
			msg.Code = proto.ProcessStatus_MAKE_BLOCK
		case types.LEAGUE_BROADCASTING:
			msg.Code = proto.ProcessStatus_LEAGUE_BROADCASTING
		case types.VOTE_START:
			msg.Code = proto.ProcessStatus_VOTE_START
		case types.VOTE_COMPLETE:
			msg.Code = proto.ProcessStatus_VOTE_COMPLETE
		case types.REQ_FAIRNODE_SIGN:
			msg.Code = proto.ProcessStatus_VOTE_COMPLETE
		case types.FINALIZE:
			msg.Code = proto.ProcessStatus_FINALIZE
		case types.REJECT:
			msg.Code = proto.ProcessStatus_REJECT
		}

		if l.Current != nil {
			msg.CurrentBlockNum = l.Current.Bytes()
		} else {
			msg.CurrentBlockNum = []byte{}
		}

		return &msg
	}

	for {
		m := makeMsg(clg) // make message
		hash := rlpHash([]interface{}{
			m.Code,
			m.CurrentBlockNum,
		})
		sign, err := rs.fn.SignHash(hash.Bytes())
		if err != nil {
			logger.Error("ProcessController signature message", "msg", err)
			return err
		}
		m.Sign = sign // add sign
		if err := stream.Send(m); err != nil {
			logger.Error("ProcessController send status message", "msg", err)
			return err
		}

		//logger.Debug("ProcessController send status message", "enode", reduceStr(nodeInfo.GetEnode()), "status", m.GetCode().String())

		if m.GetCode() == proto.ProcessStatus_REJECT {
			return errors.New(fmt.Sprintf("ProcessController league reject hash=%s", reduceStr(otprnHash.String())))
		}
		time.Sleep(1 * time.Second)
	}
}

func (rs *rpcServer) Vote(ctx context.Context, vote *proto.Vote) (*empty.Empty, error) {

	if vote.GetVoterAddress() == "" {
		return nil, errorEmpty("vote address")
	}

	if bytes.Compare(vote.GetHeader(), emptyByte) == 0 {
		return nil, errorEmpty("header")
	}

	if bytes.Compare(vote.GetVoterSign(), emptyByte) == 0 {
		return nil, errorEmpty("vote sign")
	}

	hash := rlpHash([]interface{}{
		vote.GetHeader(),
		vote.GetVoterAddress(),
	})

	addr := common.HexToAddress(vote.GetVoterAddress())
	err := verify.ValidationSignHash(vote.GetVoterSign(), hash, addr)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("sign validation failed msg=%s", err.Error()))
	}

	header := new(types.Header)
	err = rlp.DecodeBytes(vote.GetHeader(), header)
	if err != nil {
		logger.Error("vote decode header", "msg", err)
		return nil, err
	}

	otprn, err := types.DecodeOtprn(header.Otprn)
	if err != nil {
		return nil, err
	}

	err = otprn.ValidateSignature() // check otprn
	if err != nil {
		return nil, err
	}

	var l *league // current block number
	otprnHash := otprn.HashOtprn()
	if league, ok := rs.leagues[otprnHash]; ok {
		l = league
	} else {
		return nil, errors.New(fmt.Sprintf("this vote is not matched in any league hash=%s", otprnHash.String()))
	}

	if l.Current.Uint64()+1 != header.Number.Uint64() { // check block number
		return nil, errors.New(fmt.Sprintf("invalid block number current=%d vote=%d", l.Current.Uint64(), header.Number.Uint64()))
	}

	err = verify.ValidationDifficulty(header) // check block difficulty
	if err != nil {
		return nil, err
	}

	voter := types.Voter{
		Header:   vote.GetHeader(),
		Voter:    addr,
		VoteSign: vote.GetVoterSign(),
	}

	rs.db.SaveVote(otprnHash, header.Number, &voter)

	l.Mu.Lock()
	l.Voted = append(l.Voted, true) // known league
	l.Mu.Unlock()

	logger.Info("vote save", "voter", vote.GetVoterAddress(), "number", header.Number.String(), "hash", header.Hash())
	return &empty.Empty{}, nil
}

func (rs *rpcServer) RequestVoteResult(ctx context.Context, res *proto.ReqVoteResult) (*proto.ResVoteResult, error) {
	if res.GetAddress() == "" {
		return nil, errorEmpty("address")
	}

	if bytes.Compare(res.GetOtprnHash(), emptyByte) == 0 {
		return nil, errorEmpty("otprn hash")
	}

	if bytes.Compare(res.GetSign(), emptyByte) == 0 {
		return nil, errorEmpty("sign")
	}

	hash := rlpHash([]interface{}{
		res.GetOtprnHash(),
		res.GetAddress(),
	})

	addr := common.HexToAddress(res.GetAddress())
	err := verify.ValidationSignHash(res.GetSign(), hash, addr)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("sign validation failed msg=%s", err.Error()))
	}

	otprnHash := common.BytesToHash(res.GetOtprnHash())
	var clg *league // current league
	if league, ok := rs.leagues[otprnHash]; ok {
		clg = league
	} else {
		return nil, errors.New(fmt.Sprintf("this otprn is not matched in league hash=%s", res.GetOtprnHash()))
	}

	voters := rs.db.GetVoters(fairdb.MakeVoteKey(otprnHash, new(big.Int).Add(clg.Current, big.NewInt(1))))
	if len(voters) == 0 {
		return nil, errors.New("voters count is zero")
	}

	var pVoters []*proto.Vote
	for _, vote := range voters {
		// vote result validation
		hash := rlpHash([]interface{}{
			vote.Header,
			vote.Voter.String(),
		})

		err := verify.ValidationSignHash(vote.VoteSign, hash, vote.Voter)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("sign validation failed msg=%s", err.Error()))
		}

		pVoters = append(pVoters, &proto.Vote{
			Header:       vote.Header,
			VoterAddress: vote.Voter.String(),
			VoterSign:    vote.VoteSign,
		})
	}

	finalBlockHash := verify.ValidationFinalBlockHash(voters) // block hash
	voteHash := types.Voters(voters).Hash()                   // voter hash

	clg.BlockHash = &finalBlockHash
	clg.Votehash = &voteHash

	logger.Info("Request VoteResult final block", "hash", finalBlockHash, "voteHash", voteHash, "len", len(voters))

	msg := proto.ResVoteResult{
		Result:    proto.Status_SUCCESS,
		BlockHash: finalBlockHash.String(),
		Voters:    pVoters,
	}

	hash = rlpHash([]interface{}{
		msg.Result,
		msg.BlockHash,
		msg.Voters,
	})

	sign, err := rs.fn.SignHash(hash.Bytes())
	if err != nil {
		logger.Error("Request VoteResult signature message", "msg", err)
		return nil, err
	}
	msg.Sign = sign // add sign

	return &msg, nil
}

func (rs *rpcServer) SealConfirm(reqSeal *proto.ReqConfirmSeal, stream fairnode.FairnodeService_SealConfirmServer) error {
	if reqSeal.GetAddress() == "" {
		return errorEmpty("address")
	}

	if bytes.Compare(reqSeal.GetOtprnHash(), emptyByte) == 0 {
		return errorEmpty("otprn hash")
	}

	if bytes.Compare(reqSeal.GetBlockHash(), emptyByte) == 0 {
		return errorEmpty("block hash")
	}

	if bytes.Compare(reqSeal.GetVoteHash(), emptyByte) == 0 {
		return errorEmpty("vote hash")
	}

	if bytes.Compare(reqSeal.GetSign(), emptyByte) == 0 {
		return errorEmpty("sign")
	}

	hash := rlpHash([]interface{}{
		reqSeal.GetOtprnHash(),
		reqSeal.GetBlockHash(),
		reqSeal.GetVoteHash(),
		reqSeal.GetAddress(),
	})

	addr := common.HexToAddress(reqSeal.GetAddress())
	err := verify.ValidationSignHash(reqSeal.GetSign(), hash, addr)
	if err != nil {
		return errors.New(fmt.Sprintf("sign validation failed msg=%s", err.Error()))
	}

	otprnHash := common.BytesToHash(reqSeal.GetOtprnHash())
	var clg *league // current league
	if league, ok := rs.leagues[otprnHash]; ok {
		clg = league
	} else {
		return errors.New(fmt.Sprintf("this otprn is not matched in league hash=%s", reqSeal.GetOtprnHash()))
	}

	if clg.Votehash == nil || clg.BlockHash == nil {
		return errors.New("current league votehash or blockhash is nil")
	}

	voteHash := common.BytesToHash(reqSeal.GetVoteHash())
	blockHash := common.BytesToHash(reqSeal.GetBlockHash())

	if *clg.Votehash != voteHash {
		return errors.New(fmt.Sprintf("not match current vote hash=%s, req hash=%s", clg.Votehash.String(), voteHash.String()))
	}

	if *clg.BlockHash != blockHash {
		return errors.New(fmt.Sprintf("not match current block hash=%s, req hash=%s", clg.BlockHash.String(), blockHash.String()))
	}

	makeMsg := func(l *league) *proto.ResConfirmSeal {
		var m proto.ResConfirmSeal
		switch l.Status {
		case types.SEND_BLOCK:
			m.Code = proto.ProcessStatus_SEND_BLOCK
		case types.SEND_BLOCK_WAIT:
			m.Code = proto.ProcessStatus_SEND_BLOCK_WAIT
		case types.REQ_FAIRNODE_SIGN:
			m.Code = proto.ProcessStatus_REQ_FAIRNODE_SIGN
		}
		return &m
	}

	if sBlock := rs.db.GetBlock(blockHash); sBlock != nil {
		clg.Status = types.REQ_FAIRNODE_SIGN
	} else {
		clg.Status = types.SEND_BLOCK
	}

	for {
		m := makeMsg(clg)
		if m == nil {
			continue
		}

		hash = rlpHash([]interface{}{
			m.Code,
		})

		sign, err := rs.fn.SignHash(hash.Bytes())
		if err != nil {
			logger.Error("SealConfirm signature message", "msg", err)
			return err
		}
		m.Sign = sign // add sign

		if err := stream.Send(m); err != nil {
			logger.Error("SealConfirm send status message", "msg", err)
			return err
		}

		logger.Debug("SealConfirm send status message", "address", reqSeal.GetAddress(), "status", m.GetCode().String())

		if clg.Status == types.REQ_FAIRNODE_SIGN {
			// fairnode signature request signal submit and exit rpc
			return nil
		}

		time.Sleep(1 * time.Second)
	}
}

func (rs *rpcServer) SendBlock(ctx context.Context, req *proto.ReqBlock) (*empty.Empty, error) {
	if req.GetAddress() == "" {
		return nil, errorEmpty("address")
	}

	if bytes.Compare(req.GetBlock(), emptyByte) == 0 {
		return nil, errorEmpty("block")
	}

	if bytes.Compare(req.GetSign(), emptyByte) == 0 {
		return nil, errorEmpty("sign")
	}

	hash := rlpHash([]interface{}{
		req.GetBlock(),
		req.GetAddress(),
	})

	addr := common.HexToAddress(req.GetAddress())
	err := verify.ValidationSignHash(req.GetSign(), hash, addr)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("sign validation failed msg=%s", err.Error()))
	}

	block := new(types.Block)
	stream := rlp.NewStream(bytes.NewReader(req.GetBlock()), 0)
	if err := block.DecodeRLP(stream); err != nil {
		log.Error("decode block", "msg", err)
		return nil, err
	}

	otprn, err := types.DecodeOtprn(block.Otprn())
	if err != nil {
		log.Error("decode otprn", "msg", err)
		return nil, err
	}

	var clg *league // current league
	if league, ok := rs.leagues[otprn.HashOtprn()]; ok {
		clg = league
	} else {
		return nil, errors.New(fmt.Sprintf("this otprn is not matched in league hash=%s", otprn.HashOtprn().String()))
	}

	if *clg.BlockHash != block.Hash() {
		return nil, errors.New(fmt.Sprintf("not match current block hash=%s, req hash=%s", clg.BlockHash.String(), block.Hash().String()))

	}

	if *clg.Votehash != block.VoterHash() {
		return nil, errors.New(fmt.Sprintf("not match current vote hash=%s, req hash=%s", clg.Votehash.String(), block.VoterHash().String()))

	}

	clg.Status = types.SEND_BLOCK_WAIT // saving block
	err = rs.db.SaveFinalBlock(block)
	if err != nil {
		if err == fairdb.ErrAlreadyExistBlock {
			// already exist block
			clg.Status = types.REQ_FAIRNODE_SIGN
			return &empty.Empty{}, nil
		}
		clg.Status = types.SEND_BLOCK_WAIT
		return nil, err
	}

	clg.Status = types.REQ_FAIRNODE_SIGN
	return &empty.Empty{}, nil
}

func (rs *rpcServer) RequestFairnodeSign(ctx context.Context, reqInfo *proto.ReqFairnodeSign) (*proto.ResFairnodeSign, error) {
	if reqInfo.GetAddress() == "" {
		return nil, errorEmpty("address")
	}

	if bytes.Compare(reqInfo.GetOtprnHash(), emptyByte) == 0 {
		return nil, errorEmpty("otprn hash")
	}

	if bytes.Compare(reqInfo.GetBlockHash(), emptyByte) == 0 {
		return nil, errorEmpty("block hash")
	}

	if bytes.Compare(reqInfo.GetVoteHash(), emptyByte) == 0 {
		return nil, errorEmpty("vote hash")
	}

	if bytes.Compare(reqInfo.GetSign(), emptyByte) == 0 {
		return nil, errorEmpty("sign")
	}

	hash := rlpHash([]interface{}{
		reqInfo.GetOtprnHash(),
		reqInfo.GetBlockHash(),
		reqInfo.GetVoteHash(),
		reqInfo.GetAddress(),
	})

	addr := common.HexToAddress(reqInfo.GetAddress())
	err := verify.ValidationSignHash(reqInfo.GetSign(), hash, addr)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("sign validation failed msg=%s", err.Error()))
	}

	otprnHash := common.BytesToHash(reqInfo.GetOtprnHash())
	var clg *league // current league
	if league, ok := rs.leagues[otprnHash]; ok {
		clg = league
	} else {
		return nil, errors.New(fmt.Sprintf("this otprn is not matched in league hash=%s", reqInfo.GetOtprnHash()))
	}

	if clg.Votehash == nil || clg.BlockHash == nil {
		return nil, errors.New("current league votehash or blockhash is nil")
	}

	voteHash := common.BytesToHash(reqInfo.GetVoteHash())
	blockHash := common.BytesToHash(reqInfo.GetBlockHash())

	if *clg.Votehash != voteHash {
		return nil, errors.New(fmt.Sprintf("not match current vote hash=%s, req hash=%s", clg.Votehash.String(), voteHash.String()))
	}

	if *clg.BlockHash != blockHash {
		return nil, errors.New(fmt.Sprintf("not match current block hash=%s, req hash=%s", clg.BlockHash.String(), blockHash.String()))
	}

	hash = rlpHash([]interface{}{
		reqInfo.GetBlockHash(),
		reqInfo.GetVoteHash(),
	})

	signature, err := rs.fn.SignHash(hash.Bytes())
	if err != nil {
		logger.Error("Request Fairnode Signature message", "msg", err)
		return nil, err
	}

	m := proto.ResFairnodeSign{
		Signature: signature,
	}

	hash = rlpHash([]interface{}{
		m.GetSignature(),
	})

	sign, err := rs.fn.SignHash(hash.Bytes())
	if err != nil {
		logger.Error("Request Fairnode Signature message", "msg", err)
		return nil, err
	}

	m.Sign = sign

	return &m, nil
}
