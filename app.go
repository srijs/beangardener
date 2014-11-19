package main

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"net"
	"net/http"
)

func readChannel(r io.Reader, c io.Closer, chErr chan error) chan []byte {
	chBuf := make(chan []byte)
	go func() {
		buf := make([]byte, 256)
		for {
			n, err := r.Read(buf)
			if err != nil {
				chErr <- err
				break
			}
			chBuf <- buf[:n]
		}
		c.Close()
	}()
	return chBuf
}

func newProxy(addr *net.TCPAddr) func([]byte, *bufio.Reader, *net.TCPConn) {
	return func(preamble []byte, reader *bufio.Reader, conn *net.TCPConn) {
		proxyConn, err := net.DialTCP("tcp", nil, addr)
		if err != nil {
			log.Print(err)
			return
		}
		chErr := make(chan error)
		chRight := readChannel(proxyConn, conn, chErr)
		chLeft := readChannel(reader, proxyConn, chErr)
		preambleWritten := false
		halfClosed := false
		for {
			select {
			case data := <-chLeft:
				if !preambleWritten {
					proxyConn.Write(append(preamble, data...))
					preambleWritten = true
				} else {
					proxyConn.Write(data)
				}
			case data := <-chRight:
				conn.Write(data)
			case <-chErr:
				if !halfClosed {
					halfClosed = true
				} else {
					return
				}
			default:
				if !preambleWritten {
					proxyConn.Write(preamble)
					preambleWritten = true
				}
			}
		}
	}
}

func serveProxy(listener *net.TCPListener) {

	proxyBean := newProxy(&net.TCPAddr{[]byte{127, 0, 0, 1}, 11300, ""})
	proxyHttp := newProxy(&net.TCPAddr{[]byte{127, 0, 0, 1}, 8000, ""})

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Fatal(err)
		}
		go func() {
			reader := bufio.NewReader(conn)
			line, err := reader.ReadBytes('\n')
			if err != nil {
				log.Print(err)
				return
			}
			httpLineBuf := bytes.NewBuffer(line)
			_, err = httpLineBuf.WriteString("\r\n\r\n")
			if err != nil {
				log.Print(err)
				return
			}
			_, err = http.ReadRequest(bufio.NewReader(httpLineBuf))
			if err != nil {
				proxyBean(line, reader, conn)
			} else {
				proxyHttp(line, reader, conn)
			}
		}()
	}

}

func main() {

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    err := checkHealth()
    if err != nil {
      log.Print(err)
      w.WriteHeader(500)
    } else {
		  w.WriteHeader(200)
    }
	})

	log.Println("Serving HTTP...")
	go http.ListenAndServe("localhost:8000", nil)

	listener, err := net.ListenTCP("tcp", &net.TCPAddr{[]byte{0, 0, 0, 0}, 8080, ""})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Serving TCP...")
	serveProxy(listener)

}
