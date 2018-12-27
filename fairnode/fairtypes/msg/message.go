package msg

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/anduschain/go-anduschain/rlp"
	"log"
	"net"
	"time"
)

type Msg struct {
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

func ReadMsg(msg []byte) *Msg {
	m := &Msg{}
	m.ReceivedAt = time.Now()
	rlp.Decode(bytes.NewReader(msg), &m)
	return m
}

func (m Msg) Decode(val interface{}) error {
	err := rlp.Decode(bytes.NewReader(m.Payload), val)
	if err != nil {
		return err
	}

	return nil
}

func Send(msgcode uint32, data interface{}, conn net.Conn) error {
	msg, err := makeMassage(msgcode, data)
	if err != nil {
		fmt.Println("message", err)
		return err
	}

	lengthBuf := make([]byte, 8)
	binary.BigEndian.PutUint32(lengthBuf, uint32(msg.Len()))

	if _, err := conn.Write(lengthBuf); nil != err {
		log.Printf("failed to send msg length; err: %v", err)
		return err
	}

	if _, err := conn.Write(msg.Bytes()); err == nil {
		return nil
	} else {
		fmt.Println("message send", err)
		return err
	}
}

func makeMassage(msgcode uint32, data interface{}) (bytes.Buffer, error) {
	var b bytes.Buffer
	var network bytes.Buffer
	var err error

	err = rlp.Encode(&b, data)
	if err != nil {
		fmt.Println("andus >> msg.Send EncodeToBytes 에러", err)
		return network, err
	}

	err = rlp.Encode(&network, Msg{Code: msgcode, Size: uint32(b.Len()), Payload: b.Bytes()})
	if err != nil {
		fmt.Println("andus >> msg.Send EncodeToBytes 에러", err)
		return network, err
	}

	fmt.Println("make message length :", network.Len(), b.Len())

	return network, nil
}
