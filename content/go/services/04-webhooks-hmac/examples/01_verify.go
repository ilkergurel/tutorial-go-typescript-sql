package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const tolerance = 5 * time.Minute

func mac(secret []byte, ts int64, body []byte) []byte {
	m := hmac.New(sha256.New, secret)
	fmt.Fprintf(m, "%d.", ts) // the timestamp is signed too
	m.Write(body)
	return m.Sum(nil)
}

// sign is the sender's side of the scheme.
func sign(secret []byte, t time.Time, body []byte) string {
	return fmt.Sprintf("t=%d,v1=%x", t.Unix(), mac(secret, t.Unix(), body))
}

// verify accepts the body if any v1 signature matches any of our secrets.
func verify(header string, body []byte, secrets [][]byte, now time.Time) error {
	var ts int64
	var sigs [][]byte
	for part := range strings.SplitSeq(header, ",") {
		k, v, _ := strings.Cut(part, "=")
		if k == "t" {
			ts, _ = strconv.ParseInt(v, 10, 64)
		} else if b, err := hex.DecodeString(v); k == "v1" && err == nil {
			sigs = append(sigs, b)
		}
	}
	if ts == 0 || len(sigs) == 0 {
		return errors.New("malformed signature header")
	}
	if age := now.Sub(time.Unix(ts, 0)); age > tolerance || age < -tolerance {
		return errors.New("timestamp outside tolerance")
	}
	for _, secret := range secrets {
		for _, sig := range sigs {
			if hmac.Equal(sig, mac(secret, ts, body)) {
				return nil
			}
		}
	}
	return errors.New("no valid signature")
}

func main() {
	now := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	current, previous := []byte("whsec_2026_11"), []byte("whsec_2026_10")
	secrets := [][]byte{current, previous} // both accepted while the sender rotates
	paid := []byte(`{"id":"evt_1","type":"payment.succeeded","amount_kurus":125000}`)
	forged := []byte(`{"id":"evt_1","type":"payment.succeeded","amount_kurus":1}`)

	check := func(name, header string, body []byte) {
		result := "accepted"
		if err := verify(header, body, secrets, now); err != nil {
			result = "rejected: " + err.Error()
		}
		fmt.Printf("%-22s %s\n", name, result)
	}
	check("valid", sign(current, now, paid), paid)
	check("tampered body", sign(current, now, paid), forged)
	check("guessed secret", sign([]byte("whsec_guess"), now, paid), paid)
	check("replayed 10 min later", sign(current, now.Add(-10*time.Minute), paid), paid)
	check("replayed 1 min later", sign(current, now.Add(-time.Minute), paid), paid)
	check("signed with old secret", sign(previous, now, paid), paid)
	check("no header", "", paid)
}
