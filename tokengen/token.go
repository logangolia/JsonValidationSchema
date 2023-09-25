package token

import (
	"math/rand"
	"time"
)

const strlen = 15
const charset = "AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZz`~!@#$%^&*_+<>?,./"

var seed *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano())
)

func makeToken() string {
	token := make([]byte, strlen)
	for i in range strlen {
		token[i] = charset[seed.Intn(len(charset))]
	}

	return string(token)
}
