package main

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

// acceptKey proves to the client that the server speaks WebSocket: SHA-1 of
// the client's key plus a GUID fixed by RFC 6455, in base64.
func acceptKey(key string) string {
	sum := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func ws(w http.ResponseWriter, r *http.Request) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		w.Header().Set("Upgrade", "websocket") // a 426 must name the protocol
		http.Error(w, "WebSocket upgrade required", http.StatusUpgradeRequired)
		return
	}
	// A real server checks r.Header.Get("Origin") here: browsers don't apply
	// CORS to WebSockets, so any web page could otherwise connect.
	conn, rw, err := http.NewResponseController(w).Hijack()
	if err != nil {
		http.Error(w, "cannot take over the connection", http.StatusInternalServerError)
		return
	}
	defer conn.Close()
	fmt.Fprintf(rw, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\n"+
		"Connection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", acceptKey(r.Header.Get("Sec-WebSocket-Key")))

	// No longer HTTP: frames. One unmasked text frame, FIN bit + opcode 1;
	// the second byte is the payload length in bytes (here under 126).
	msg := "Merhaba dünya"
	rw.Write([]byte{0x81, byte(len(msg))})
	rw.WriteString(msg)
	rw.Flush()
}

func main() {
	srv := httptest.NewServer(http.HandlerFunc(ws))
	defer srv.Close()

	plain, _ := http.Get(srv.URL)
	plain.Body.Close()
	fmt.Println("plain GET:", plain.Status, "Upgrade:", plain.Header.Get("Upgrade"))

	const key = "dGhlIHNhbXBsZSBub25jZQ==" // the sample nonce from RFC 6455
	req, _ := http.NewRequest("GET", srv.URL, nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", key)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer resp.Body.Close() // after a 101, Body is the raw two-way connection
	fmt.Println("upgrade:  ", resp.Status)
	fmt.Println("accept:   ", resp.Header.Get("Sec-WebSocket-Accept"), resp.Header.Get("Sec-WebSocket-Accept") == acceptKey(key))

	head := make([]byte, 2)
	io.ReadFull(resp.Body, head)
	payload := make([]byte, head[1]&0x7f)
	io.ReadFull(resp.Body, payload)
	fmt.Printf("frame:     FIN=%t opcode=%d length=%d bytes text=%q (%d characters)\n",
		head[0]&0x80 != 0, head[0]&0x0f, len(payload), payload, len([]rune(string(payload))))
}
