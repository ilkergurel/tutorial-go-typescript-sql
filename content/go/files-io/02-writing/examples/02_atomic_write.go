package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// writeInPlace truncates the target first: a failure halfway leaves half a file.
func writeInPlace(path string, write func(io.Writer) error) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return write(f)
}

// writeAtomic writes a temp file next to the target, then renames it over the
// target. Until the rename, the old content is untouched.
func writeAtomic(path string, perm os.FileMode, write func(io.Writer) error) (err error) {
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			f.Close()
			os.Remove(f.Name()) // don't leave a half-written temp file behind
		}
	}()
	if err = write(f); err != nil {
		return err
	}
	if err = f.Chmod(perm); err != nil { // CreateTemp made it 0o600
		return err
	}
	if err = f.Sync(); err != nil { // data on disk before the new name is
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func main() {
	for _, name := range []string{"inplace.json", "atomic.json"} {
		os.WriteFile(name, []byte(`{"version":1,"city":"İzmir"}`), 0o644)
	}
	crash := func(w io.Writer) error {
		io.WriteString(w, `{"version":2,"ci`)
		return errors.New("simulated crash") // the process dies here
	}
	fmt.Println("in place:", writeInPlace("inplace.json", crash))
	fmt.Println("atomic:  ", writeAtomic("atomic.json", 0o644, crash))

	for _, name := range []string{"inplace.json", "atomic.json"} {
		data, _ := os.ReadFile(name)
		fmt.Printf("%-12s valid JSON: %-5t %s\n", name, json.Valid(data), data)
	}
	temps, _ := filepath.Glob("*.tmp-*")
	fmt.Println("temp files left:", len(temps))
}
