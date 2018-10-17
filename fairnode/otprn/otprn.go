package otprn

type Otprn struct {
}

func New() (*Otprn, error) {

	return &Otprn{}, nil
}

func (otprn *Otprn) CheckOtprn(aa string) (*Otprn, error) {

	//TODO : andus >> 서명값 검증, otrprn 구조체 검증

	return &Otprn{}, nil
}
