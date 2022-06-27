package mock_fairnode

//go:generate mockgen -source=$PWD/protos/fairnode/fairnode.pb.go -destination=$PWD/fairnode/mock_fairnode/mock_fairnode.go
//go:generate mockgen -source=$PWD/../../protos/fairnode/fairnode.pb.go -destination=$PWD/mock_fairnode.go
//go:generate mockgen -source=$PWD/../../protos/fairnode/fairnode_grpc.pb.go -destination=$PWD/mock_fairnode_grpc.go
import (
	"context"
	"github.com/anduschain/go-anduschain/protos/common"
	"github.com/anduschain/go-anduschain/protos/fairnode"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes/empty"
	"testing"
	"time"
)

func TestHeartBeat(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := NewMockFairnodeServiceClient(ctrl)
	msg := common.HeartBeat{}
	client.EXPECT().HeartBeat(gomock.Any(), &msg).Return(&empty.Empty{}, nil)
	testHeartBeat(t, client)
}

func testHeartBeat(t *testing.T, client fairnode.FairnodeServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := client.HeartBeat(ctx, &common.HeartBeat{})
	if err != nil {
		t.Errorf("mocking failed %t", err)
	}
	t.Log("Reply : OK ")
}
