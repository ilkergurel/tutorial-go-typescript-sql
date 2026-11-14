package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
)

const maxUpload = 64 << 10 // 64 KiB per request, for the demo

// upload reads the parts one at a time and streams each file into a hash:
// the file is never held in memory, and net/http writes no temp files.
func upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "expected multipart/form-data", http.StatusUnsupportedMediaType)
		return
	}
	var report strings.Builder // answer after the body is read, never during
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			http.Error(w, "upload larger than 64 KiB", http.StatusRequestEntityTooLarge)
			return
		}
		if err != nil {
			http.Error(w, "malformed multipart body", http.StatusBadRequest)
			return
		}
		sum := sha256.New()
		n, err := io.Copy(sum, part) // a file part or a plain form field
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			http.Error(w, "upload larger than 64 KiB", http.StatusRequestEntityTooLarge)
			return
		}
		fmt.Fprintf(&report, "part %q filename=%q: %d bytes, sha256 %x...\n",
			part.FormName(), part.FileName(), n, sum.Sum(nil)[:4])
	}
	io.WriteString(w, report.String())
}

// send streams a multipart body from a goroutine through a pipe, so the
// client doesn't buffer the upload either.
func send(url, fileName string, size int) {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		mw.WriteField("title", "Ekim raporu")
		fw, _ := mw.CreateFormFile("file", fileName)
		io.Copy(fw, strings.NewReader(strings.Repeat("x", size)))
		pw.CloseWithError(mw.Close())
	}()
	resp, err := http.Post(url, mw.FormDataContentType(), pr)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("%d\n%s", resp.StatusCode, body)
}

func main() {
	srv := httptest.NewServer(http.HandlerFunc(upload))
	defer srv.Close()
	send(srv.URL, "rapor-İş.csv", 10_000)
	send(srv.URL, "../../etc/passwd", 10)
	send(srv.URL, "big.bin", 200_000)
}
