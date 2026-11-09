package utils

import (
	"fmt"
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

func RunPprof(host string) {
	go func() {
		fmt.Println("Run Pprof")
		fmt.Println(http.ListenAndServe(host, nil))
	}()
}
