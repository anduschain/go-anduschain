package msg

import (
	"net"
)

type Msg struct {
}

const (
	ReqLeagueJoinOK = iota
	ResLeagueJoinTrue
	ResLeagueJoinFalse
	SendEnode
	SendOTPRN
)

func ReadMsg(conn net.Conn) *Msg {

	return &Msg{}
}

func Send(msgcode uint64, data interface{}, conn net.Conn) error {
	return nil
}
