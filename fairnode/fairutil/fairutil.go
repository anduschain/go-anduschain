package fairutil

func IsJoinOK() bool {
	//TODO : andus >> 참여자 여부 계산

	return false
}

// geth Node IP를 key로 하는 Peer map

func GetPeerList() map[string][]string {

	// TODO : andus >> 피어 리스트 만들어 주는 함수..

	val := make(map[string][]string)

	// TODO : andus >> reutrn Data sampel

	val["192.168.0.1"] = []string{"enode://010a0098320943280x:123.123.0.1:30002", "enode://010a0098320943280x:123.123.0.1:30002"}

	return val
}
