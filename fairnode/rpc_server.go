package fairnode

import (
	"bytes"
	"context"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairdb"
	"github.com/anduschain/go-anduschain/fairnode/verify"
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

	if block := rs.db.CurrentBlock(); block != nil {
		if block.Hash().String() != nodeInfo.GetHead() {
			return nil, errors.New("head is mismatch")
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

			logger.Info("otprn submitted", "otrpn", otprn.HashOtprn().String(), "enode", reduceStr(nodeInfo.GetEnode()))
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

	// TODO(hakuna) : self를 제외한 enodeurl 전송 구현 해야됨
	var enodes []string
	for _, node := range nodes {
		enodes = append(enodes, node.EnodeUrl())
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

	var status *types.FnStatus // 해당되는 리그의 상태
	var currnet *big.Int       // current block number
	otprnHash := common.BytesToHash(nodeInfo.GetOtprnHash())
	if league, ok := rs.leagues[otprnHash]; ok {
		status = &league.Status
		currnet = league.Current
	} else {
		return errors.New(fmt.Sprintf("this otprn is not matched in league hash=%s", nodeInfo.GetOtprnHash()))
	}

	makeMsg := func(status *types.FnStatus, currentNum []byte) *proto.ProcessMessage {
		var msg proto.ProcessMessage
		switch *status {
		case types.SAVE_OTPRN:
			msg.Code = proto.ProcessStatus_WAIT
		case types.MAKE_LEAGUE:
			msg.Code = proto.ProcessStatus_MAKE_LEAGUE
		case types.MAKE_JOIN_TX:
			msg.Code = proto.ProcessStatus_MAKE_JOIN_TX
		case types.MAKE_BLOCK:
			msg.Code = proto.ProcessStatus_MAKE_BLOCK
		case types.VOTE_START:
			msg.Code = proto.ProcessStatus_VOTE_START
		case types.VOTE_COMPLETE:
			msg.Code = proto.ProcessStatus_VOTE_COMPLETE
		}
		msg.CurrentBlockNum = currentNum
		return &msg
	}

	for {
		m := makeMsg(status, currnet.Bytes()) // make message
		hash := rlpHash([]interface{}{
			m.Code,
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
		logger.Debug("ProcessController send status message", "enode", reduceStr(nodeInfo.GetEnode()), "status", m.GetCode().String())
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

	var currnet *big.Int // current block number
	otprnHash := otprn.HashOtprn()
	if league, ok := rs.leagues[otprnHash]; ok {
		currnet = league.Current
	} else {
		return nil, errors.New(fmt.Sprintf("this vote is not matched in any league hash=%s", otprnHash.String()))
	}

	if currnet.Uint64()+1 != header.Number.Uint64() { // check block number
		return nil, errors.New(fmt.Sprintf("invalid block number current=%d vote=%d", currnet.Uint64(), header.Number.Uint64()))
	}

	err = verify.ValidationDifficulty(header) // check block difficulty
	if err != nil {
		return nil, err
	}

	voter := types.Voter{}

	rs.db.SaveVote(otprnHash, header.Number, voter)

	return &empty.Empty{}, nil
}

func (rs *rpcServer) RequestVoteResult(ctx context.Context, res *proto.ReqVoteResult) (*proto.ResVoteResult, error) {
	return nil, nil
}

func (rs *rpcServer) SealConfirm(reqSeal *proto.ReqConfirmSeal, stream fairnode.FairnodeService_SealConfirmServer) error {
	return nil
}

func (rs *rpcServer) SendBlock(ctx context.Context, block *proto.ReqBlock) (*empty.Empty, error) {
	return nil, nil
}

func (rs *rpcServer) RequestFairnodeSign(ctx context.Context, reqInfo *proto.ReqFairnodeSign) (*proto.ResFairnodeSign, error) {
	return nil, nil
}
