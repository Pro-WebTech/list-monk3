package secure

import (
	"fmt"
	"hash"
	"strconv"
	"time"

	"github.com/nbutton23/zxcvbn-go"
)

// New initalizes security service
func New(minPWStr int, h hash.Hash) *Service {
	return &Service{minPWStr: minPWStr, h: h}
}

// Service holds security related methods
type Service struct {
	minPWStr int
	h        hash.Hash
}

// Password checks whether password is secure enough using zxcvbn library
func (s *Service) Password(pass string, inputs ...string) bool {
	pwStrength := zxcvbn.PasswordStrength(pass, inputs)
	return pwStrength.Score >= s.minPWStr
}

// Token generates new unique token
func (s *Service) Token(str string) string {
	s.h.Reset()
	fmt.Fprintf(s.h, "%s%s", str, strconv.Itoa(time.Now().Nanosecond()))
	return fmt.Sprintf("%x", s.h.Sum(nil))
}
