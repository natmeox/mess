package main

import (
    "sync"
    "github.com/jameskeane/bcrypt"
    "log"
    uuid "github.com/nu7hatch/gouuid"
)

type Account struct {
    AccountName string
    passwordHash string
    objectId uuid.UUID
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

func LoadAccount(accountName string) (account *Account) {
    rows, err := db.Query("select accountName, passwordHash, objectId from account where accountName = ?",
        accountName)
    if err != nil {
        return
    }

    var passwordHash string
    var objectId []byte
    for rows.Next() {
        rows.Scan(&accountName, &passwordHash, &objectId)
        id, _ := uuid.Parse(objectId)
        account = &Account{accountName, passwordHash, *id}
        accounts[accountName] = account
    }
    return
}

func (account *Account) Save() {
    _, err := db.Exec("insert or replace into account (accountName, passwordHash, objectId) values (?, ?, ?)",
        account.AccountName, account.passwordHash, account.objectId[0:16])
    if err != nil {
        log.Println("Could not save account", account.AccountName, ":", err.Error())
    }
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

    go account.Save()
    return &account, nil
}

func VerifyAccount (accountName string, password string) (*Account, error) {
    accountsLock.Lock()
    defer accountsLock.Unlock()

    account := accounts[accountName]  // or nil?
    if account == nil {
        account = LoadAccount(accountName)
    }
    if account == nil || account.AccountName != accountName {
        return nil, AuthError("No such account.")
    }

    if !account.HasPassword(password) {
        return nil, AuthError("Not the right password.")
    }

    return account, nil
}
