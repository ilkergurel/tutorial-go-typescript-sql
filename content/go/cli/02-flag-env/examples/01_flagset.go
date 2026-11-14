package main

import (
	"errors"
	"flag"
	"fmt"
	"time"
)

type Config struct {
	Verbose bool
	Timeout time.Duration
	Retries int
	Files   []string
}

// parse uses its own FlagSet instead of the global flag.Parse, so a test can
// call it; ContinueOnError makes Parse return errors instead of exiting.
func parse(args []string) (Config, error) {
	var c Config
	fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
	fs.BoolVar(&c.Verbose, "v", false, "log every request")
	fs.DurationVar(&c.Timeout, "timeout", 5*time.Second, "per-request timeout")
	fs.IntVar(&c.Retries, "retries", 3, "attempts after the first failure")
	if err := fs.Parse(args); err != nil {
		return c, err
	}
	c.Files = fs.Args() // whatever is left after the flags
	return c, nil
}

func main() {
	for _, args := range [][]string{
		{},
		{"-v", "--timeout=1.5s", "-retries", "0", "a.txt", "b.txt"},
		{"a.txt", "-v"},
		{"-v=false", "--", "-odd-name.txt"},
		{"-retries", "many"},
		{"-x"},
		{"-h"},
	} {
		c, err := parse(args)
		switch {
		case errors.Is(err, flag.ErrHelp):
			fmt.Printf("%q -> help requested, exit 0\n", args)
		case err != nil:
			fmt.Printf("%q -> %v, exit 2\n", args, err)
		default:
			fmt.Printf("%q -> %+v\n", args, c)
		}
	}
}
