package main

import (
	"bufio"
	"fmt"
	"net"
)

func checkHealth() error {

	addr := &net.TCPAddr{[]byte{127, 0, 0, 1}, 11300, ""}
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(conn)

	_, err = conn.Write([]byte("use health\r\n"))
	if err != nil {
		return err
	}

	line, _, err := reader.ReadLine()
	if err != nil {
		return err
	}

	if string(line) != "USING health" {
		return fmt.Errorf("Malformed response to use: %s", line)
	}

	_, err = conn.Write([]byte("put 0 0 60 6\r\nfoobar\r\n"))
	if err != nil {
		return err
	}

	line, _, err = reader.ReadLine()
	if err != nil {
		return err
	}

	var jobId int
	_, err = fmt.Sscanf(string(line), "INSERTED %d", &jobId)
	if err != nil {
		return fmt.Errorf("Malformed response to reserve: %s", line)
	}

	_, err = conn.Write([]byte("watch health\r\n"))
	if err != nil {
		return err
	}

	line, _, err = reader.ReadLine()
	if err != nil {
		return err
	}

	var watchCount int
	_, err = fmt.Sscanf(string(line), "WATCHING %d", &watchCount)
	if err != nil {
		return fmt.Errorf("Malformed response to reserve: %s", line)
	}

	_, err = conn.Write([]byte("reserve\r\n"))
	if err != nil {
		return err
	}

	line, _, err = reader.ReadLine()
	if err != nil {
		return err
	}

	var reserveId int
	_, err = fmt.Sscanf(string(line), "RESERVED %d", &reserveId)
	if err != nil {
		return fmt.Errorf("Malformed response to reserve: %s", line)
	}

	line, _, err = reader.ReadLine()
	if err != nil {
		return err
	}

	if string(line) != "foobar" {
		return fmt.Errorf("Malformed response to payload: %s", line)
	}

	_, err = conn.Write([]byte(fmt.Sprintf("delete %d\r\n", reserveId)))
	if err != nil {
		return err
	}

	line, _, err = reader.ReadLine()
	if err != nil {
		return err
	}

	if string(line) != "DELETED" {
		return fmt.Errorf("Malformed response to delete: %s", line)
	}

	conn.Close()

	return nil

}
