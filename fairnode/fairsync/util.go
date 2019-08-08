package fairsync

import (
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/crypto/sha3"
	"github.com/anduschain/go-anduschain/rlp"
	"net"
	"time"
)

func strToFnType(t string) FnType {
	switch t {
	case "PENDING":
		return PENDING
	case "LEADER":
		return FN_LEADER
	case "FOLLOWER":
		return FN_FOLLOWER
	default:
		return 100
	}
}

func parseNode(nodes map[common.Hash]map[string]string) map[common.Hash]fnNode {
	fns := make(map[common.Hash]fnNode)
	for id, node := range nodes {
		fns[id] = fnNode{
			ID:      id,
			Address: node["ADDRESS"],
			Status:  strToFnType(node["STATUS"]),
		}
	}
	return fns
}

func getFairnodeID() (*common.Hash, string) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logger.Error("Get Fairnode ID, ip address", "msg", err)
		return nil, ""
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				hash := rlpHash([]interface{}{
					ipnet.IP.String(),
					time.Now().String(),
				})
				return &hash, ipnet.IP.String()
			}
		}
	}

	return nil, ""
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}
