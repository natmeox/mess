package mess

import (
	"github.com/bmizerany/pq"
	"github.com/jameskeane/bcrypt"
	"log"
	"time"
)

type AccountStore interface {
	AccountForLogin(name, password string) (acc *Account)
	CreateAccount(name, password string) (acc *Account)
	GetAccount(name string) (acc *Account)
}

type Account struct {
	LoginName    string
	PasswordHash string
	Character    ThingId
	Created      time.Time
}

func (w *DatabaseWorld) GetAccount(name string) (acc *Account) {
	acc = &Account{}
	row := w.db.QueryRow("SELECT loginname, character, created FROM account WHERE loginname = $1",
		name)
	err := row.Scan(&acc.LoginName, &acc.Character, &acc.Created)
	if err != nil {
		log.Println("Error loading account with name", name, ":", err)
		return nil
	}
	return
}

func (w *DatabaseWorld) AccountForLogin(name, password string) (acc *Account) {
	acc = &Account{}
	row := w.db.QueryRow("SELECT loginname, passwordhash, character, created FROM account WHERE loginname = $1",
		name)
	err := row.Scan(&acc.LoginName, &acc.PasswordHash, &acc.Character, &acc.Created)
	// TODO: oh look there are timing attacks wheeeeeeee
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

func (w *DatabaseWorld) CreateAccount(name, password string) (acc *Account) {
	passwordHash, err := bcrypt.Hash(password)
	if err != nil {
		log.Println("Couldn't hash password to create an account:", err.Error())
		return nil
	}

	origin := World.ThingForId(1)
	char := World.CreateThing(name, origin, origin)
	if char == nil {
		log.Println("Couldn't create character to create an account")
		return nil
	}

	tx, err := w.db.Begin()
	if err != nil {
		log.Println("Couldn't open transaction to create an account:", err.Error())
		return nil
	}

	acc = &Account{name, passwordHash, char.Id, time.Unix(0, 0)}

	row := tx.QueryRow("INSERT INTO account (loginname, passwordhash, character) VALUES ($1, $2, $3) RETURNING created",
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
