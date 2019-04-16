package transport

import (
	"bytes"
	"encoding/binary"
	"github.com/anduschain/go-anduschain/rlp"
	log "gopkg.in/inconshreveable/log15.v2"
	"io"
	"net"
	"sync"
	"time"
)

const (
	frameReadTimeout  = 30 * time.Second
	frameWriteTimeout = 30 * time.Second
)

type Transport interface {
	MsgReadWriter
	Close()
}

type Tsp struct {
	fd       net.Conn
	rmu, wmu sync.Mutex
	rw       *tspRW
}

func New(fd net.Conn) Transport {
	return &Tsp{
		fd: fd,
		rw: newTspRw(fd),
	}
}

func (t *Tsp) ReadMsg() (*TsMsg, error) {
	t.rmu.Lock()
	defer t.rmu.Unlock()
	t.fd.SetReadDeadline(time.Now().Add(frameReadTimeout))
	return t.rw.ReadMsg()
}

func (t *Tsp) WriteMsg(msg *TsMsg) error {
	t.wmu.Lock()
	defer t.wmu.Unlock()
	//t.fd.SetWriteDeadline(time.Now().Add(frameWriteTimeout))
	return t.rw.WriteMsg(msg)
}

func (t *Tsp) Close() {
	t.wmu.Lock()
	defer t.wmu.Unlock()
	err := t.fd.Close()
	if err != nil {
		log.Error("transport.Close", "error", err)
	}
}

type MsgReader interface {
	ReadMsg() (*TsMsg, error)
}

type MsgWriter interface {
	WriteMsg(*TsMsg) error
}

type MsgReadWriter interface {
	MsgReader
	MsgWriter
}

func Send(w MsgWriter, code uint32, data interface{}) error {
	m, err := MakeTsMsg(code, data)
	if err != nil {
		log.Error("transport.Send > MakeTsMsg", "error", err)
	}
	err = w.WriteMsg(m)
	if err != nil {
		log.Error("transport.Send > WriteMsg", "error", err)
		return err
	}

	return nil
}

type tspRW struct {
	conn io.ReadWriter
}

func newTspRw(conn io.ReadWriter) *tspRW {
	return &tspRW{
		conn: conn,
	}
}

func (tr *tspRW) ReadMsg() (*TsMsg, error) {
	var err error
	lengthBuf := make([]byte, 8)
	_, err = tr.conn.Read(lengthBuf)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, nil
		} else {
			return nil, err
		}
	}

	msgLength := binary.BigEndian.Uint32(lengthBuf)

	receive := make([]byte, 4096)
	var buf bytes.Buffer
	for msgLength > 0 {
		n, err := tr.conn.Read(receive)
		if err != nil {
			return nil, err
		}
		if n > 0 {
			buf.Write(receive[:n])
			msgLength -= uint32(n)
		}
	}

	m := &TsMsg{}
	m.ReceivedAt = time.Now()
	err = rlp.Decode(bytes.NewReader(buf.Bytes()), &m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (tr *tspRW) WriteMsg(msg *TsMsg) error {
	var network bytes.Buffer
	err := rlp.Encode(&network, msg)
	if err != nil {
		return err
	}
	lengthBuf := make([]byte, 8)
	binary.BigEndian.PutUint32(lengthBuf, uint32(network.Len()))
	if _, err := tr.conn.Write(lengthBuf); nil != err {
		return err
	}

	if _, err := tr.conn.Write(network.Bytes()); nil != err {
		return err
	}

	return nil
}
