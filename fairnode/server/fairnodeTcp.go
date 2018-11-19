package server

import (
	"fmt"
	"github.com/anduschain/go-anduschain/common"
	"github.com/anduschain/go-anduschain/fairnode/client"
	"github.com/anduschain/go-anduschain/fairnode/fairutil"
	"github.com/anduschain/go-anduschain/rlp"
)

func (f *FairNode) ListenTCP() error {

	fmt.Println("andus >> ListenTCP 리슨TCP")

	buf := make([]byte, 4096)
	for {
		conn, err := f.TcpConn.Accept()
		if err != nil {
			fmt.Println("andus >> f.TcpConn.Accept 에러!!", err)
		}
		conn.Read(buf)
		var fromGeth fairnodeclient.TransferCheck
		rlp.DecodeBytes(buf, &fromGeth)

		fmt.Println(fromGeth.Enode.ID)

		conn.Write([]byte("넌 들어올 자격이 있어 ㅎㅎㅎ"))
	}

	//go f.sendLeague()
	//
	//// TODO : andus >> 위닝블록이 수신되는 곳 >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
	//// TODO : andus >> 1. 수신 블록 검증 ( sig, hash )
	//// TODO : andus >> 2. 검증된 블록을 MongoDB에 저장 ( coinbase, RAND, 보낸 놈, 블록 번호 )
	//// TODO : andus >> 3. 기다림........
	//// TODO : andus >> 3-1 채굴참여자 수 만큼 투표가 진행 되거나, 아니면 15초후에 투표 종료
	//
	//count := 0
	//leagueCount := 10 // TODO : andus >> 리그 채굴 참여자
	//endOfLeagueCh := make(chan interface{})
	//
	//if count == leagueCount {
	//	endOfLeagueCh <- "보내.."
	//}
	//
	//t := time.NewTicker(15 * time.Second)
	//log.Println(" @ go func() START !! ")
	//
	//go func() {
	//	for {
	//		select {
	//		case <-t.C:
	//			// TODO : andus >> 투표 결과 서명해서, TCP로 보내준다
	//			// TODO : andus >> types.TransferBlock{}의 타입으로 전송할것..
	//			// TODO : andus >> 받은 블록의 블록헤더의 해시를 이용해서 서명후, FairNodeSig에 넣어서 보낼것.
	//			// TODO : andus >> 새로운 리그 시작
	//			f.LeagueRunningOK = false
	//
	//		case <-endOfLeagueCh:
	//			// TODO : andus >> 투표 결과 서명해서, TCP로 보내준다
	//			// TODO : andus >> types.TransferBlock{}의 타입으로 전송할것..
	//			// TODO : andus >> 받은 블록의 블록헤더의 해시를 이용해서 서명후, FairNodeSig에 넣어서 보낼것.
	//			f.LeagueRunningOK = false
	//
	//		}
	//	}
	//}()

	return nil
}

func (f *FairNode) sendLeague() {
	for {
		<-f.startSignalCh

		fmt.Println("andus >> @@@ sendLeague")
		// TODO : andus >> 1. 채굴참여자 조회 ( from DB )
		// TODO : andus >> 2. 채굴 리그 구성

		//fromGeth := make([]byte, 4096)
		//
		//for {
		//	gethConn, err := f.TcpConn.Accept()
		//	if err != nil {
		//		fmt.Println("andus >> GethConn 에러", err)
		//	}
		//
		//	// otprn, address,
		//	n, err := gethConn.Read(fromGeth)
		//}

		var league []map[string]string

		leagueHash := f.makeHash(league) // TODO : andsu >> 전체 채굴리그의 해시값

		for key, value := range fairutil.GetPeerList() {
			//key = to,
			//value = 접속할 peer list

			fmt.Println(leagueHash, key, value)
			// TODO : andus >> 각 GETH 노드에게 연결할 peer 리스트 전달 + 전체 채굴리그의 해시값 ( leagueHash )
			// TODO : andus >> 추후 서명 예정....
		}
	}
}

func (f *FairNode) makeHash(list []map[string]string) common.Hash {

	return common.Hash{}
}
