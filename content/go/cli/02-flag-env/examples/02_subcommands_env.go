package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
)

var commands = map[string]func(unicode.SpecialCase, string) string{
	"upper": strings.ToUpperSpecial,
	"lower": strings.ToLowerSpecial,
}

var languages = map[string]unicode.SpecialCase{"en": nil, "tr": unicode.TurkishCase}

// run dispatches on the first argument, git-style. getenv is a parameter:
// main passes os.LookupEnv, a test passes a map.
func run(args []string, getenv func(string) (string, bool), stdout, stderr io.Writer) int {
	if len(args) == 0 || commands[args[0]] == nil {
		fmt.Fprintln(stderr, "usage: casing upper|lower [-lang en|tr] WORD...")
		return 2
	}
	lang := "en" // default < environment < flag
	if v, ok := getenv("CASING_LANG"); ok && v != "" {
		lang = v
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&lang, "lang", lang, "casing rules: en or tr (env CASING_LANG)")
	if err := fs.Parse(args[1:]); err != nil {
		return 2 // flag has already printed the error and the usage
	}
	rules, ok := languages[lang]
	if !ok {
		fmt.Fprintf(stderr, "casing %s: unknown language %q\n", args[0], lang)
		return 2
	}
	var out []string
	for _, w := range fs.Args() {
		out = append(out, commands[args[0]](rules, w))
	}
	fmt.Fprintln(stdout, strings.Join(out, " "))
	return 0
}

func show(getenv func(string) (string, bool), args ...string) {
	fmt.Printf("$ casing %s\n", strings.Join(args, " "))
	fmt.Printf("  exit %d\n", run(args, getenv, os.Stdout, os.Stderr))
}

func main() {
	env := map[string]string{"CASING_LANG": "tr"}
	fake := func(k string) (string, bool) { v, ok := env[k]; return v, ok }

	// The environment says tr; in the second call the flag overrides it.
	show(fake, "lower", "İZMİR", "IŞIK")
	show(fake, "lower", "-lang", "en", "İZMİR", "IŞIK")
	show(fake, "upper", "-lang=de", "istanbul")
	show(fake, "title", "x")

	os.Setenv("CASING_LANG", "tr") // the real environment, this process only
	show(os.LookupEnv, "upper", "istanbul", "ırmak")
}
