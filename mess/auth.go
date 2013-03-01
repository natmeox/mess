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
	row := Db.QueryRow("SELECT loginname, passwordhash, character, created FROM account WHERE loginname = ?",
		name)
	err := row.Scan(&acc)
	if err != nil {
		thing, ok := err.(*pq.SimplePGError)
		if ok {
			log.Println("OHAI A SIMPLER ERR:", thing.Error())
		} else {
			log.Println("Error loading account with name", name, ":", err.Error())
		}
		return nil
	}

	if !bcrypt.Match(password, acc.PasswordHash) {
		log.Println("Bad login attempt for account", name)
		return nil
	}

	return
}
