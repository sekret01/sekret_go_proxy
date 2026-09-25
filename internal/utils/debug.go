package utils

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"
)

func RunGorutineCounter() {
	go func() {
		for {
			fmt.Printf("[##] goroutines: %d\n", runtime.NumGoroutine())
			time.Sleep(5 * time.Second)
		}
	}()
}

func RunPprof() {
	go func() {
		log.Println(http.ListenAndServe("127.0.0.1:6060", nil))
	}()
}
