package main

import (
	"bufio"
	"bytes"
	"log"
	"net"
	"net/http"
)

func proxy(preamble []byte, reader *bufio.Reader, conn *net.TCPConn, addr *net.TCPAddr) {
	proxyConn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		log.Print(err)
		return
	}
	chLeft := make(chan []byte)
	chRight := make(chan []byte)
	chErr := make(chan error)
	go func() {
		buf := make([]byte, 256)
		for {
			n, err := proxyConn.Read(buf)
			if err != nil {
				chErr <- err
				break
			}
			chRight <- buf[:n]
		}
		conn.Close()
	}()
	go func() {
		buf := make([]byte, 256)
		for {
			n, err := reader.Read(buf)
			if err != nil {
				chErr <- err
				break
			}
			chLeft <- buf[:n]
		}
		proxyConn.Close()
	}()
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

func serve(conn *net.TCPConn) {
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
		proxy(line, reader, conn, &net.TCPAddr{[]byte{127, 0, 0, 1}, 11300, ""})
	} else {
		proxy(line, reader, conn, &net.TCPAddr{[]byte{127, 0, 0, 1}, 8000, ""})
	}
}

func main() {

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	log.Println("Serving HTTP...")
	go http.ListenAndServe("localhost:8000", nil)

	_, err := net.ResolveTCPAddr("tcp", "localhost:8080")
	if err != nil {
		log.Fatal(err)
	}

	listener, err := net.ListenTCP("tcp", &net.TCPAddr{[]byte{0, 0, 0, 0}, 8080, ""})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Serving TCP...")
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Fatal(err)
		}
		go serve(conn)
	}

}
