package main

import (
    "sync"
    "github.com/jameskeane/bcrypt"
)

type Account struct {
    AccountName string
    passwordHash string
    objectId string
}

var accounts map[string] *Account = make(map[string] *Account)
var accountsLock sync.Mutex

func (account *Account) HasPassword (password string) bool {
    return bcrypt.Match(password, account.passwordHash)
}

type AuthError string

func (err AuthError) Error() string {
    return string(err)
}

func RegisterAccount (accountName string, password string) (*Account, error) {
    passwordHash, err := bcrypt.Hash(password)
    if err != nil {
        return nil, AuthError("Your password could not be hashed (oops?): " + err.Error())
    }
    player := NewPlayer(accountName)
    account := Account{accountName, passwordHash, player.id}

    accountsLock.Lock()
    defer accountsLock.Unlock()

    existingAccount := accounts[accountName]
    if existingAccount != nil && existingAccount.AccountName == accountName {
        return nil, AuthError("That name is in use.")
    }
    accounts[accountName] = &account

    return &account, nil
}

func VerifyAccount (accountName string, password string) (*Account, error) {
    accountsLock.Lock()
    defer accountsLock.Unlock()

    account := accounts[accountName]  // or nil?
    if account == nil || account.AccountName != accountName {
        return nil, AuthError("No such account.")
    }

    if !account.HasPassword(password) {
        return nil, AuthError("Not the right password.")
    }

    return account, nil
}
