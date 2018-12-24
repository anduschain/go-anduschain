package msg

import (
	"bytes"
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
	ReqLeagueJoinOK = iota << 1
	ResLeagueJoinFalse
	SendEnode
	SendOTPRN
	SendLeageNodeList
	MakeJoinTx
	MakeBlock
	MinerLeageStop
	SendBlockForVote
	SendFinalBlock
)

func ReadMsg(msg []byte) *Msg {
	var m Msg
	m.ReceivedAt = time.Now()
	rlp.Decode(bytes.NewReader(msg), &m)
	return &m
}

func (m Msg) Decode(val interface{}) error {
	err := rlp.Decode(bytes.NewReader(m.Payload), val)
	if err != nil {
		return err
	}

	return nil
}

func Send(msgcode uint64, data interface{}, conn net.Conn) error {
	msg, err := makeMassage(msgcode, data)
	if err != nil {
		fmt.Println("message", err)
		return err
	}

	if _, err := conn.Write(msg); err == nil {
		return nil
	} else {
		fmt.Println("message send", err)
		return err
	}
}

func makeMassage(msgcode uint64, data interface{}) ([]byte, error) {
	var b bytes.Buffer
	var err error
	err = rlp.Encode(&b, data)
	if err != nil {
		fmt.Println("andus >> msg.Send EncodeToBytes 에러", err)
		return []byte{}, err
	}

	var bb bytes.Buffer
	err = rlp.Encode(&bb, Msg{Code: msgcode, Size: uint32(b.Len()), Payload: b.Bytes()})
	if err != nil {
		fmt.Println("andus >> msg.Send EncodeToBytes 에러", err)
		return []byte{}, err
	}

	fmt.Printf("make message length : %d", bb.Len())

	return bb.Bytes(), nil
}
