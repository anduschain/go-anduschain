package msg

import (
	"fmt"
	"github.com/anduschain/go-anduschain/rlp"
	"net"
	"time"
)

type Msg struct {
	Code       uint64
	Size       uint32
	Payload    []byte
	ReceivedAt time.Time
}

const (
	ReqLeagueJoinOK = iota
	ResLeagueJoinTrue
	ResLeagueJoinFalse
	SendEnode
	SendOTPRN
	SendLeageNodeList
)

func ReadMsg(msg []byte) *Msg {
	var m Msg
	m.ReceivedAt = time.Now()
	rlp.DecodeBytes(msg, &m)
	return &m
}

func (m Msg) Decode(val interface{}) error {
	return rlp.DecodeBytes(m.Payload, val)
}

func Send(msgcode uint64, data interface{}, conn net.Conn) error {
	msg, err := makeMassage(msgcode, data)
	if err != nil {
		return err
	}

	if _, err := conn.Write(msg); err != nil {
		return nil
	} else {
		return err
	}
}

func makeMassage(msgcode uint64, data interface{}) ([]byte, error) {
	bData, err := rlp.EncodeToBytes(data)
	if err != nil {
		fmt.Println("andus >> msg.Send EncodeToBytes 에러", err)
		return []byte{}, err
	}

	msg, err := rlp.EncodeToBytes(Msg{Code: msgcode, Size: uint32(len(bData)), Payload: bData})
	if err != nil {
		fmt.Println("andus >> msg.Send EncodeToBytes 에러", err)
		return []byte{}, err
	}

	return msg, nil
}
