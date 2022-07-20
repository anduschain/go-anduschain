package common

import "strings"

func SplitEnode(enode string) (string, string, string) {
	slice := strings.Split(enode, "@")
	if len(slice) != 2 {
		return "", "", ""
	}
	url := slice[0]
	slice2 := strings.Split(slice[1], ":")
	if len(slice2) != 2 {
		return "", "", ""
	}
	host := slice2[0]
	port := slice2[1]

	return url, host, port
}
