package mess

import (
	"github.com/bmizerany/pq"
	"github.com/jameskeane/bcrypt"
	"log"
	"time"
)

type Account struct {
	LoginName    string
	PasswordHash string
	Character    int
	Created      time.Time
}

func AccountForLogin(name, password string) (acc *Account) {
	acc = &Account{}
	row := Db.QueryRow("SELECT loginname, passwordhash, character, created FROM account WHERE loginname = $1",
		name)
	err := row.Scan(&acc.LoginName, &acc.PasswordHash, &acc.Character, &acc.Created)
	if err != nil {
		thing, ok := err.(pq.PGError)
		var message string
		if ok {
			message = thing.Get('M')
		} else {
			message = err.Error()
		}
		log.Println("Error loading account with name", name, ":", message)
		return nil
	}

	if !bcrypt.Match(password, acc.PasswordHash) {
		log.Println("Bad login attempt for account", name)
		return nil
	}

	return
}

func CreateAccount(name, password string) (acc *Account) {
	passwordHash, err := bcrypt.Hash(password)
	if err != nil {
		log.Println("Couldn't hash password to create an account:", err.Error())
		return nil
	}

	tx, err := Db.Begin()
	if err != nil {
		log.Println("Couldn't open transaction to create an account:", err.Error())
		return nil
	}

	acc = &Account{name, passwordHash, 0, time.Unix(0, 0)}
	row := tx.QueryRow("INSERT INTO character (name, description) VALUES ($1, $2) RETURNING id",
		name, "")
	err = row.Scan(&acc.Character)
	if err != nil {
		log.Println("Couldn't create character for new account:", err.Error())
		tx.Rollback()
		return nil
	}

	row = tx.QueryRow("INSERT INTO account (loginname, passwordhash, character) VALUES ($1, $2, $3) RETURNING created",
		name, passwordHash, acc.Character)
	err = row.Scan(&acc.Created)
	if err != nil {
		log.Println("Couldn't create new account:", err.Error())
		tx.Rollback()
		return nil
	}

	err = tx.Commit()
	if err != nil {
		log.Println("Couldn't commit transaction to create new account:", err.Error())
		tx.Rollback()
		return nil
	}

	return
}
