// LEARN(design.patterns.anti-patterns): getters, setters and a rule in a service vs a type that keeps its invariant.
package main

import (
	"errors"
	"fmt"
)

// Java-shaped: an interface for the only implementation, Get/Set pairs, and
// the rule kept in a service that any caller can walk around.
type IAccount interface {
	GetBalance() int
	SetBalance(int)
}

type accountImpl struct{ balance int }

func (a *accountImpl) GetBalance() int  { return a.balance }
func (a *accountImpl) SetBalance(b int) { a.balance = b }

type AccountService struct{}

func (AccountService) Withdraw(a IAccount, amount int) error {
	if a.GetBalance() < amount {
		return errors.New("insufficient funds")
	}
	a.SetBalance(a.GetBalance() - amount)
	return nil
}

// Go-shaped: the type owns its rule. The getter is Balance, not GetBalance,
// and there is no setter, because no caller should set a balance.
type Account struct{ balance int }

var ErrInsufficient = errors.New("insufficient funds")

func (a *Account) Balance() int { return a.balance }

func (a *Account) Withdraw(amount int) error {
	if amount <= 0 {
		return fmt.Errorf("withdraw %d: amount must be positive", amount)
	}
	if a.balance < amount {
		return ErrInsufficient
	}
	a.balance -= amount
	return nil
}

func show(who string, err error, balance int) {
	fmt.Printf("%-8s balance=%-4d err=%v\n", who, balance, err)
}

func main() {
	var anemic IAccount = &accountImpl{balance: 100}
	err := AccountService{}.Withdraw(anemic, -400) // a negative withdrawal deposits
	show("anemic", err, anemic.GetBalance())
	anemic.SetBalance(-50) // or skip the service altogether
	show("anemic", nil, anemic.GetBalance())

	acc := &Account{balance: 100}
	for _, amount := range []int{-400, 150, 30} {
		err := acc.Withdraw(amount)
		show("account", err, acc.Balance())
	}
}
