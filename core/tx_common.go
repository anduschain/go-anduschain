package core

type transactionSet struct {
	Join *txList
	Gen  *txList
}

func (ts *transactionSet) Len() int {
	return ts.Join.Len() + ts.Gen.Len()
}
