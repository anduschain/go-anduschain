package fairnode

import (
	"bytes"
	"context"
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/core/types"
	"github.com/anduschain/go-anduschain/fairnode/fairdb"
	proto "github.com/anduschain/go-anduschain/protos/common"
	"github.com/anduschain/go-anduschain/protos/fairnode"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"math/big"
	"strings"
	"time"
)

var (
	emptyByte []byte
)

func errorEmpty(key string) error {
	return errors.New(fmt.Sprintf("%s value is empty", key))
}

// fairnode rpc method implemented
type rpcServer struct {
	db fairdb.FairnodeDB
}

func newServer(db fairdb.FairnodeDB) *rpcServer {
	return &rpcServer{
		db: db,
	}
}

// Heart Beat : notify to fairnode, I'm alive.
func (rs *rpcServer) HeartBeat(ctx context.Context, nodeInfo *proto.HeartBeat) (*empty.Empty, error) {
	if nodeInfo.GetEnode() == "" {
		return nil, errorEmpty("enode")
	}

	if nodeInfo.GetChainID() == "" {
		return nil, errorEmpty("chainID")
	}

	if nodeInfo.GetMinerAddress() == "" {
		return nil, errorEmpty("miner's address")
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
		nodeInfo.GetHead(),
	})

	err := ValidationSignHash(nodeInfo.GetSign(), hash, common.HexToAddress(nodeInfo.GetMinerAddress()))
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
		Head:         common.HexToHash(nodeInfo.GetHead()),
		Time:         big.NewInt(time.Now().Unix()),
	})

	defer logger.Info("HeartBeat received", "enode", nodeInfo.Enode)
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

	err := ValidationSignHash(nodeInfo.GetSign(), hash, addr)
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
		return nil, errors.New(fmt.Sprintf("does not exist in active node list addr = %s", nodeInfo.GetEnode()))
	}

	if otprn := rs.db.CurrentOtprn(); otprn != nil {
		err := otprn.ValidateSignature()
		if err != nil {
			return nil, errors.New(fmt.Sprintf("otprn validation failed msg=%s", err.Error()))
		}

		// 참가 대상 확인
		if IsJoinOK(otprn, addr) {
			// OTPRN 전송
			bOtprn, err := otprn.EncodeOtprn()
			if err != nil {
				return nil, errors.New(fmt.Sprintf("otprn EncodeOtprn failed msg=%s", err.Error()))
			}
			logger.Info("otprn submitted", "otrpn", otprn.HashOtprn().String())
			return &proto.ResOtprn{
				Result: proto.Status_SUCCESS,
				Otprn:  bOtprn,
			}, nil
		} else {
			return &proto.ResOtprn{
				Result: proto.Status_SUCCESS,
			}, nil
		}
	} else {
		return nil, errors.New("otprn is not exist")
	}
}

func (rs *rpcServer) ProcessController(nodeInfo *proto.Participate, stream fairnode.FairnodeService_ProcessControllerServer) error {
	logger.Info("ProcessController", "message", nodeInfo.Enode)
	return nil
}

func (rs *rpcServer) Vote(ctx context.Context, vote *proto.Vote) (*empty.Empty, error) {
	return nil, nil
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
