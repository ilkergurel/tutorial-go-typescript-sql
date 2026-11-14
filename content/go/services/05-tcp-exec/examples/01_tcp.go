package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
	"unicode"
)

// serve upper-cases every line it receives, Turkish rules, one goroutine
// per connection.
func serve(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return // the listener was closed
		}
		go func() {
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(5 * time.Second)) // no client keeps us forever
			sc := bufio.NewScanner(conn)
			for sc.Scan() {
				fmt.Fprintln(conn, strings.ToUpperSpecial(unicode.TurkishCase, sc.Text()))
			}
		}()
	}
}

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:0") // port 0: the OS picks a free one
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	go serve(ln)

	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.DialContext(context.Background(), "tcp", ln.Addr().String())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	words := []string{"istanbul", "ıspanak", "çiğdem"}
	io.WriteString(conn, strings.Join(words, "\n")+"\n") // one Write, three messages
	conn.(*net.TCPConn).CloseWrite()                     // "that's all": the server's Scan ends
	sc := bufio.NewScanner(conn)
	for i := 0; sc.Scan(); i++ {
		fmt.Printf("%-8s %d bytes -> %-8s %d bytes\n", words[i], len(words[i]), sc.Text(), len(sc.Text()))
	}
	if err := sc.Err(); err != nil {
		log.Fatal(err)
	}

	quiet, err := d.DialContext(context.Background(), "tcp", ln.Addr().String())
	if err != nil {
		log.Fatal(err)
	}
	defer quiet.Close()
	quiet.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, err = quiet.Read(make([]byte, 1)) // we sent nothing, so nothing comes back
	fmt.Println("read with nothing to read: deadline exceeded =", errors.Is(err, os.ErrDeadlineExceeded))
}
