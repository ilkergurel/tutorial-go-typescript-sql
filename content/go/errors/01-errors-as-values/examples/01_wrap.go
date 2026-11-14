package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

func loadConfig(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		// %w embeds the cause; %v would format the text and drop the chain.
		return nil, fmt.Errorf("load config: %w", err)
	}
	return data, nil
}

func main() {
	if _, err := loadConfig("users.txt"); err != nil {
		fmt.Println(err)

		// errors.Is unwraps: fs.ErrNotExist matches through both wrappers.
		fmt.Println("not exist?", errors.Is(err, fs.ErrNotExist))
	}
}
