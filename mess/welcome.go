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

func WelcomeClient(client *ClientPump) {
	screen, err := loadWelcomeScreen()
	if err != nil {
		screen = "WELCOME"
	}

	client.ToClient <- screen
	for input := range client.ToServer {
		parts := strings.SplitN(input, " ", 2)
		command := strings.ToLower(parts[0])
		if command == "connect" {
			name := parts[1]
			// hurf
			client.ToClient <- "YAY CONNECTED AS " + name + " okay bye"
			close(client.ToClient)
			return
		}

		client.ToClient <- screen
	}
}
