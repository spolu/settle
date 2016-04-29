package randstr

import (
	"strings"
	"testing"
)

func TestCardNumber(t *testing.T) {
	number := CardNumber("123456")
	if !strings.HasPrefix(number, "123456") {
		t.Errorf("card number has wrong prefix: %s", number)
	} else if len(number) != 16 {
		t.Errorf("card number has wrong length: %s", number)
	} else if !luhn(number) {
		t.Errorf("card number is not Luhn valid: %s", number)
	}
}
