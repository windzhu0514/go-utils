package main

import (
	"log"
	"time"

	"github.com/windzhu0514/go-utils/tcpool"
)

func main() {
	client()
}

func client() {
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
			log.Printf("ConnectStart network: %s addr:%s err:%v", network, addr, err)
		},
	}
	p := tcpool.New("localhost:9999", tcpool.WithTrace(trace), tcpool.WithMaxBadConnRetries(1))
	defer p.Close()

	t := time.NewTicker(time.Second * 5)
	for range t.C {
		p.Write([]byte("test.debug\n"))
		// p.Write([]byte("test.info\n"))
		// p.Write([]byte("test.warn\n"))
		// p.Write([]byte("test.error\n"))
	}
}
