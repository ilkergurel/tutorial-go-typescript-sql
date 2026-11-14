package main

import (
	"fmt"
	"strconv"
	"time"
)

type Celsius float64
type Fahrenheit float64

func toF(c Celsius) Fahrenheit { return Fahrenheit(c*9/5 + 32) }

func main() {
	boiling := Celsius(100)
	fmt.Println(toF(boiling))
	// var f Fahrenheit = boiling // compile error: both are float64 underneath,
	// but the named type carries the unit and Go won't mix them silently

	retries := 3
	backoff := time.Duration(retries) * time.Second // int * Duration needs a conversion
	fmt.Println(backoff)
	fmt.Println(time.Duration(2)) // a bare 2 means 2 nanoseconds

	// Numbers to text go through strconv; string(rune(n)) is a code point.
	fmt.Println(strconv.Itoa(65), string(rune(65)))
	port, err := strconv.Atoi("80a")
	fmt.Println(port, err)

	// strconv is locale-blind: a Turkish decimal comma is a syntax error.
	_, err = strconv.ParseFloat("3,14", 64)
	fmt.Println(err)
}
