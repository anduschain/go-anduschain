package main

import (
	"github.com/anduschain/go-anduschain/fairnode/db"
	"log"
)

func main() {
	// TODO : andus >> cli 프로그램에서 환경변수 및 운영변수를 세팅 할 수 있도록 구성...
	_, err := db.New()

	if err != nil {
		log.Fatal(err)
	}

	// TODO : db 연결 테스트
	// TODO : UDP Listen PORT : 60002
	// TODO : TCP Listen PORT : 60001

}
