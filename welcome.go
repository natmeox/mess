package mess

import (
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path"
	"strings"
	"time"
)

func loadWelcomeScreen() (string, error) {
	now := time.Now()

	// Is there one for the date and year?
	welcomepath := path.Join("welcome", "date", now.Format("20060102"))
	fi, err := os.Stat(welcomepath)
	if err != nil {
		// Is there one for just the date?
		welcomepath = path.Join("welcome", "date", now.Format("0102"))
		fi, err = os.Stat(welcomepath)
		if err != nil {
			welcomepath = "welcome"
			fi, err = os.Stat(welcomepath)
			if err != nil {
				log.Println("Could not find candidate welcome screens anywhere:", err)
				return "", err
			}
		}
	}

	// If it's already a file, use it.
	if !fi.IsDir() {
		welcomescreen, err := ioutil.ReadFile(welcomepath)
		if err != nil {
			log.Println("Could not read new welcome screen", welcomepath, ":", err)
			return "", err
		}
		return string(welcomescreen), nil
	}

	allFileinfos, err := ioutil.ReadDir(welcomepath)
	if err != nil {
		log.Println("Could not select welcome screen from directory", welcomepath, ":", err)
		return "", err
	}

	for {
		numFileinfos := len(allFileinfos)
		if numFileinfos == 0 {
			log.Println("Found no readable welcome screens in directory", welcomepath)
			break
		}

		selection := rand.Intn(numFileinfos)
		fi = allFileinfos[selection]

		selectedpath := path.Join(welcomepath, fi.Name())
		screen, err := ioutil.ReadFile(selectedpath)
		if err == nil {
			return string(screen), nil
		}

		allFileinfos[selection] = allFileinfos[numFileinfos-1]
		allFileinfos = allFileinfos[:numFileinfos-1]
	}

	return "", err
}

func WelcomeConnect(client *ClientPump, rest string) (endWelcome bool) {
	parts := strings.SplitN(rest, " ", 2)
	if len(parts) < 2 {
		client.Send("To connect, type: connect name password")
		return false
	}
	name, password := parts[0], parts[1]

	// TODO: eventually connections should be made through a front-end that talks to a service, so the service can be restarted independently of the front-end. This would be very different.

	account := Accounts.AccountForLogin(name, password)
	if account == nil {
		client.Send("Hmm, there doesn't appear to be an account with that name and password.")
		return false
	}

	log.Println("Someone connected as", account.LoginName, "!")

	// TODO: start up the game routine
	go GameClient(client, account)

	return true
}

func WelcomeRegister(client *ClientPump, rest string) {
	parts := strings.SplitN(rest, " ", 2)
	if len(parts) < 2 {
		client.Send("To register, type: register name password")
		return
	}
	name, password := parts[0], parts[1]

	account := Accounts.CreateAccount(name, password)
	if account == nil {
		client.Send("Oops, we were unable to register you with that name.")
		return
	}

	client.Send("Yay, you were successfully registered. Type 'connect name password' to connect!")
}

func WelcomeClient(client *ClientPump) {
	screen, err := loadWelcomeScreen()
	if err != nil {
		screen = "WELCOME"
	}

	client.Send(screen)
	for input := range client.ToServer {
		if input == "QUIT" {
			client.Send("Thanks for spending time with the mess today!")
			client.Close()
			return
		}

		parts := strings.SplitN(input, " ", 2)
		command := strings.ToLower(parts[0])
		rest := ""
		if len(parts) > 1 {
			rest = parts[1]
		}

		switch command {
		case "connect":
			if WelcomeConnect(client, rest) {
				return
			}
		case "register":
			WelcomeRegister(client, rest)
		default:
			client.Send(screen)
		}
	}
}
