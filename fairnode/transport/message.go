package transport

import (
	"bytes"
	"fmt"
	"github.com/anduschain/go-anduschain/rlp"
	"net"
	"time"
)

type TsMsg struct {
	Code       uint32
	Size       uint32
	Payload    []byte
	ReceivedAt time.Time
}

const (
	ReqLeagueJoinOK = iota
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

func (m *TsMsg) Decode(val interface{}) error {
	err := rlp.Decode(bytes.NewReader(m.Payload), val)
	if err != nil {
		return err
	}

	return nil
}

func (m *TsMsg) EncodeToByte() ([]byte, error) {
	var network bytes.Buffer
	err := rlp.Encode(&network, m)
	if err != nil {
		return nil, err
	}

	return network.Bytes(), nil
}

func SendUDP(msgcode uint32, data interface{}, conn net.Conn) error {
	msg, err := MakeTsMsg(msgcode, data)
	if err != nil {
		fmt.Println("message", err)
		return err
	}

	byteMsg, err := msg.EncodeToByte()
	if err != nil {
		return err
	}

	if _, err := conn.Write(byteMsg); err == nil {
		return nil
	} else {
		fmt.Println("message send", err)
		return err
	}
}

func ReadUDP(msg []byte) *TsMsg {
	m := &TsMsg{}
	m.ReceivedAt = time.Now()
	err := rlp.Decode(bytes.NewReader(msg), &m)
	if err != nil {
		return nil
	}
	return m
}

func MakeTsMsg(msgcode uint32, data interface{}) (*TsMsg, error) {
	var b bytes.Buffer
	err := rlp.Encode(&b, data)
	if err != nil {
		fmt.Println("EncodeToBytes 에러", err)
		return nil, err
	}

	return &TsMsg{Code: msgcode, Size: uint32(b.Len()), Payload: b.Bytes()}, nil
}
