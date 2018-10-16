package fairnode

import (
	"errors"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"time"
)

var (
	makeOtprnError = errors.New("OTPRN 구조체 생성 오류")
)

type FairNode struct {
	enode         *string
	listenUDPport string
	listenTCPPort string
	otprn         *otprn.Otprn
}

func New() (*FairNode, error) {

	// TODO : andus >> otprn 구조체 생성
	otprn, err := otprn.New()
	if err != nil {
		return nil, makeOtprnError
	}

	return &FairNode{
		otprn: otprn,
	}, nil
}

func (f *FairNode) ListenUDP() error {

	startSignal := make(chan string)

	go f.manageActiveNode(startSignal)

	go f.makeLeague(startSignal)

	return nil
}

func (f *FairNode) ListenTCP() error {

	return nil
}

func (f *FairNode) manageActiveNode(aa chan string) {
	// TODO : andus >> Geth node Heart beat update ( Active node 관리 )

	t := time.NewTicker(3 * time.Second)

	for {
		select {
		case <-t.C:
			// TODO : andus >> Active Node 카운트 최초 3개 이상 ( 에러 처리 해야함.... )
			// Start signal <-
		}
	}
}

func (f *FairNode) makeLeague(aa chan string) {
	for {
		select {}
		// <- chan Start singnal
		// TODO : andus >> 리그 스타트 ( 엑티브 노드 조회 ) - OTPRN 생성 - 배포
		// close(Start singnal)
	}
}
