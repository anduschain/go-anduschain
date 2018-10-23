package fairnode

import (
	"errors"
	"github.com/anduschain/go-anduschain/fairnode/fairutil"
	"github.com/anduschain/go-anduschain/fairnode/otprn"
	"time"
)

var (
	makeOtprnError = errors.New("OTPRN 구조체 생성 오류")
)

type FairNode struct {
	enode           *string
	listenUDPport   string
	listenTCPPort   string
	otprn           *otprn.Otprn
	SingedOtprn     *string // 전자서명값
	startSignal     chan string
	startMakeLeague chan string
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

	go f.manageActiveNode(f.startSignal)

	go f.makeLeague(f.startSignal, f.startMakeLeague)

	return nil
}

func (f *FairNode) ListenTCP() error {

	// TODO : andus >> 1. 접속한 GETH노드의 채굴 참여 가능 여부 확인 ( 참여 검증 )
	_ := fairutil.IsJoinOK()
	// TODO : andus >> 참여자 별로 가능여부 체크 후, 불가한 노드는 disconnect

	// TODO : andus >> 2. 채굴 가능 노드들의 enode값 저장

	go f.sendLeague(f.startMakeLeague)

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

func (f *FairNode) makeLeague(aa chan string, bb chan string) {
	for {
		select {}
		// <- chan Start singnal // 레그 스타트

		// TODO : andus >> 리그 스타트 ( 엑티브 노드 조회 ) ->

		// TODO : andus >> 1. OTPRN 생성
		// TODO : andus >> 2. OTPRN Hash
		// TODO : andus >> 3. Fair Node 개인키로 암호화
		// TODO : andus >> 4. OTPRN 값 + 전자서명값 을 전송
		// TODO : andus >> 5. UDP 전송
		// TODO : andus >> 6. UDP 전송 후 참여 요청 받을 때 까지 기다릴 시간( 3s )후
		// TODO : andus >> 7. 리스 시작 채널에 메세지 전송
		bb <- "리그시작"

		// close(Start singnal)
	}
}

func (f *FairNode) sendLeague(aa chan string) {
	for {
		<-aa
		// TODO : andus >> 1. 채굴참여자 조회 ( from DB )
		// TODO : andus >> 2. 채굴 리그 구성

		for key, value := range fairutil.GetPeerList() {
			_ := key
			_ := value
			//key = to,
			//value = 접속할 peer list
			// TODO: andus >> 각 GETH 노드에게 연결할 peer 리스트 전달
			// TODO: andus >> 추후 서명 예정....
		}
	}
}
