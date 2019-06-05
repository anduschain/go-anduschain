package transaction

import "errors"

var (
	ErrInvalidSig = errors.New("invalid transaction v, r, s values")
)
