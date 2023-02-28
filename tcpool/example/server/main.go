package main

import (
	"bufio"
	"log"
	"net"
)

func main() {
	server()
}

func server() {
	l, err := net.Listen("tcp", "localhost:9999")
	if err != nil {
		log.Println(err)
		return
	}

	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			return
		}

		go func(c net.Conn) {
			defer c.Close()

			r := bufio.NewReader(c)
			for {
				line, isPrefix, err := r.ReadLine()
				if err != nil {
					log.Println(err)
					c.Close()
					return
				}

				// n, err := c.Read(bytes)
				// if err != nil {
				// 	log.Println(err)
				// 	return
				// }
				log.Println(string(line), isPrefix)
			}
		}(conn)
	}
}
