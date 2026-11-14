package main

import (
	"context"
	"fmt"
)

// An unexported key type: no other package can build this key, so none can
// read or overwrite the value by accident.
type ctxKey int

const requestIDKey ctxKey = 0

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestID hides the `any` and the type assertion behind a typed accessor.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id // "" when absent
}

// The user ID is an input, so it is a parameter. The request ID is
// request-scoped metadata for logs, so it may ride in the context.
func findUser(ctx context.Context, userID int) {
	fmt.Printf("[%s] SELECT name FROM users WHERE id = %d\n", RequestID(ctx), userID)
}

func main() {
	ctx := WithRequestID(context.Background(), "req-7f3a")
	findUser(ctx, 42)
	findUser(context.Background(), 43) // missing value: zero value, no panic

	// Keys compare by type and value. Two packages that both pick the
	// text "user" don't collide as long as each uses its own key type.
	type authKey string
	type logKey string
	ctx = context.WithValue(ctx, authKey("user"), "ayşe")
	ctx = context.WithValue(ctx, logKey("user"), "audit-bot")
	fmt.Println(ctx.Value(authKey("user")), ctx.Value(logKey("user")))
	fmt.Println(ctx.Value("user")) // a plain string is yet another key
}
