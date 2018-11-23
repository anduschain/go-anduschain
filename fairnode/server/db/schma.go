package db

import "time"

type activeNode struct {
	EnodeId  string
	Coinbase string
	Ip       string
	Time     time.Time
}
