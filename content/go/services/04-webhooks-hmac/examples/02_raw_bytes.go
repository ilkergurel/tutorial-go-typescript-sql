package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

func mac(secret, msg []byte) []byte {
	m := hmac.New(sha256.New, secret)
	m.Write(msg)
	return m.Sum(nil)
}

func main() {
	secret := []byte("whsec_2026_11")
	// What a Python sender's json.dumps produces: non-ASCII escaped as \uXXXX.
	raw := []byte(`{"type":"order.created","customer":"Ay\u015fe \u00c7elik","note":"<VIP> & co"}`)
	sig := mac(secret, raw) // the header the sender attached

	// The tempting shortcut: decode first, re-encode, verify that.
	var ev map[string]any
	if err := json.Unmarshal(raw, &ev); err != nil {
		panic(err)
	}
	reencoded, _ := json.Marshal(ev)

	fmt.Printf("received:   %s\n", raw)
	fmt.Printf("re-encoded: %s\n", reencoded)
	fmt.Println("same JSON value, same bytes?", string(raw) == string(reencoded))
	fmt.Println("verify the received bytes: ", hmac.Equal(sig, mac(secret, raw)))
	fmt.Println("verify the re-encoded bytes:", hmac.Equal(sig, mac(secret, reencoded)))

	name := ev["customer"].(string)
	fmt.Printf("customer %q: %d characters, %d bytes\n", name, len([]rune(name)), len(name))
}
