package main

import (
	"cmp"
	"fmt"
	"time"
)

type Client struct {
	baseURL string
	timeout time.Duration
	retries int
}

func (c *Client) String() string {
	return fmt.Sprintf("%s timeout=%s retries=%d", c.baseURL, c.timeout, c.retries)
}

// Option is a functional option: a function that adjusts the Client being built.
type Option func(*Client)

func WithTimeout(d time.Duration) Option { return func(c *Client) { c.timeout = d } }

func WithRetries(n int) Option { return func(c *Client) { c.retries = n } }

// New sets the defaults first, then lets each option override one of them.
func New(baseURL string, opts ...Option) *Client {
	c := &Client{baseURL: baseURL, timeout: 10 * time.Second, retries: 2}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Config is the plainer alternative: a struct whose zero fields mean "default".
type Config struct {
	Timeout time.Duration // 0 means 10s
	Retries int           // 0 means 2, so "no retries" can't be expressed
}

func NewFromConfig(baseURL string, cfg Config) *Client {
	return &Client{
		baseURL: baseURL,
		timeout: cmp.Or(cfg.Timeout, 10*time.Second), // first non-zero value
		retries: cmp.Or(cfg.Retries, 2),
	}
}

func main() {
	const runner = "http://127.0.0.1:8090"
	fmt.Println(New(runner))
	fmt.Println(New(runner, WithTimeout(45*time.Second)))
	fmt.Println(New(runner, WithRetries(0))) // an explicit zero is kept

	fmt.Println(NewFromConfig(runner, Config{Timeout: 45 * time.Second}))
	fmt.Println(NewFromConfig(runner, Config{Retries: 0})) // asked for 0, got 2
}
