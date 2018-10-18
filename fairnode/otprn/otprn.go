package otprn

type Otprn struct {
	rand      int64
	Cminer    int64
	Mminer    int64
	TimeStamp string
	S         string
	R         string
	V         string
}

func New() (*Otprn, error) {

	return &Otprn{}, nil
}

func (otprn *Otprn) CheckOtprn(aa string) (*Otprn, error) {

	//TODO : andus >> 서명값 검증, otrprn 구조체 검증

	return &Otprn{}, nil
}

func (otprn *Otprn) SignOtprn() {

}

func (otprn *Otprn) HashOtprn() {

}

// TODO : andus >> 1. OTPRN 생성
// TODO : andus >> 2. OTPRN Hash
// TODO : andus >> 3. Fair Node 개인키로 암호화
// TODO : andus >> 4. OTPRN 값 + 전자서명값 을 전송
