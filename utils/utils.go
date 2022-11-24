package utils

import (
	"log"
	"os"
)

func Debug(l ...interface{}) {
	if os.Getenv("DEBUG") == "1" {
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)

		prefix := "[shadiaosocketio]"

		log.Println(append([]interface{}{prefix}, l...)...)
	}
}
