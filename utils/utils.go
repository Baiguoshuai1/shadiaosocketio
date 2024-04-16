package utils

import (
	jsoniter "github.com/json-iterator/go"
	"log"
	"os"
	"unicode/utf8"
)

var Json = func() jsoniter.API {
	// https://github.com/json-iterator/go/issues/244
	jsoniter.RegisterExtension(&BinaryAsStringExtension{})
	return jsoniter.ConfigCompatibleWithStandardLibrary
}()

func Debug(l ...interface{}) {
	if os.Getenv("DEBUG") == "1" {
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)

		prefix := "[shadiaosocketio]"

		log.Println(append([]interface{}{prefix}, l...)...)
	}
}

// equalASCIIFold returns true if s is equal to t with ASCII case folding as
// defined in RFC 4790.
func EqualASCIIFold(s, t string) bool {
	for s != "" && t != "" {
		sr, size := utf8.DecodeRuneInString(s)
		s = s[size:]
		tr, size := utf8.DecodeRuneInString(t)
		t = t[size:]
		if sr == tr {
			continue
		}
		if 'A' <= sr && sr <= 'Z' {
			sr = sr + 'a' - 'A'
		}
		if 'A' <= tr && tr <= 'Z' {
			tr = tr + 'a' - 'A'
		}
		if sr != tr {
			return false
		}
	}
	return s == t
}
