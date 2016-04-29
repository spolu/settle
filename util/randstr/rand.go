package randstr

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
)

// Tokens

const (
	tokenLength = 16
	a62         = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789__"
)

var tokens = make(tokenFountain, 512)

type tokenFountain chan string

func (f tokenFountain) Write(
	buf []byte,
) (int, error) {
	var token [tokenLength]byte
	var i int
	for _, b := range buf {
		if b != '_' {
			token[i] = b
			i++
		}
		if i == tokenLength {
			f <- string(token[:])
			i = 0
		}
	}

	return len(buf), nil
}

// Cards

var (
	cardNumbers = int64(1000000000) // 10**9
	cards       = make(chan int64, 512)
)

func init() {
	buf := bufio.NewWriterSize(tokens, 1024)
	enc := base64.NewEncoder(base64.NewEncoding(a62), buf)

	go func() {
		_, err := io.Copy(enc, rand.Reader)
		// If rand.Reader ever ends or throws an error, we're
		// going to have a bad time, and there's really not
		// much we can do about it.
		log.Panicln("utils.rand: token creation ran out of entropy", err)
	}()

	go func() {
		for {
			number, err := rand.Int(rand.Reader, big.NewInt(cardNumbers))
			if err != nil {
				log.Panicln(
					"utils.rand: card number creation ran out of entropy", err)
			}
			cards <- number.Int64()
		}
	}()
}

// Token generates a random token prefixed by prefix
func Token(
	prefix string,
) string {
	return prefix + "_" + Str()
}

// Str generates a random string
func Str() string {
	return <-tokens
}

// CardNumber generates a random card number
func CardNumber(
	bin string,
) string {
	number := <-cards
	for i := 0; i < 10; i++ {
		candidate := fmt.Sprintf("%s%010d", bin, 10*number+int64(i))
		if luhn(candidate) {
			return candidate
		}
	}

	panic(errors.New("CardNumber: code should be unreachable"))
}

func luhn(
	s string,
) bool {
	t := [...]int{0, 2, 4, 6, 8, 1, 3, 5, 7, 9}
	odd := len(s) & 1
	var sum int
	for i, c := range s {
		if c < '0' || c > '9' {
			return false
		}
		if i&1 == odd {
			sum += t[c-'0']
		} else {
			sum += int(c - '0')
		}
	}
	return sum%10 == 0
}
