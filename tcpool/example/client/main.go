package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/windzhu0514/go-utils/tcpool"
)

func main() {
	client()
}

func client() {
	go func() {
		http.ListenAndServe(":9998", nil)
	}()

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	trace := &tcpool.ConnTrace{
		GetConn: func(hostPort string) { log.Println("GetConn: " + hostPort) },
		GotConn: func(info tcpool.GotConnInfo) { log.Printf("GotConn: %+v", info) },
		PutIdleConn: func(err error) {
			log.Printf("PutIdleConn: %v\n", err)
		},
		ConnectStart: func(network, addr string) {
			log.Printf("ConnectStart network: %s addr:%s", network, addr)
		},
		ConnectDone: func(network, addr string, err error) {
			log.Printf("ConnectDone network: %s addr:%s err:%v", network, addr, err)
		},
	}
	p := tcpool.New("localhost:9999", tcpool.Option{Trace: trace, MaxConns: 1})
	defer p.Close()

	p.Write([]byte("begin \n"))
	// t := time.NewTicker(time.Hour * 5)
	t := time.NewTicker(time.Second * 10)
	for range t.C {
		msg := "test.debug " + time.Now().String()
		fmt.Println(msg)
		p.Write([]byte(msg + "\n"))
		// p.Write([]byte("test.info\n"))
		// p.Write([]byte("test.warn\n"))
		// p.Write([]byte("test.error\n"))
	}
}
