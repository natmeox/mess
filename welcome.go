package main

import (
    "container/list"
    "io/ioutil"
    "log"
    "math/rand"
    "os"
    "path"
    "strings"
    "time"
)

type welcomeScreen struct {
    Screens chan string
    screenText string
    loadAfter time.Time
}

var WelcomeScreen welcomeScreen

func NewWelcomeScreen() {
    WelcomeScreen = welcomeScreen{make(chan string), "~WELCOME~", time.Unix(0, 0)}
}

func (screen *welcomeScreen) chooseScreen() {
    now := time.Now()

    // Is there one for the date and year?
    welcomepath := path.Join("welcome", "date", now.Format("20060102"))
    fi, error := os.Stat(welcomepath)
    if error != nil {
        // Is there one for just the date?
        welcomepath = path.Join("welcome", "date", now.Format("0102"))
        fi, error = os.Stat(welcomepath)
        if error != nil {
            welcomepath = "welcome"
            fi, error = os.Stat(welcomepath)
            if error != nil {
                log.Println("Could not find candidate welcome screens anywhere:", error)
                return
            }
        }
    }

    // If it's already a file, use it.
    if !fi.IsDir() {
        welcomescreen, error := ioutil.ReadFile(welcomepath)
        if error != nil {
            log.Println("Could not read new welcome screen", welcomepath, ":", error)
            return
        }
        screen.screenText = string(welcomescreen)
        return
    }

    allFileinfos, error := ioutil.ReadDir(welcomepath)
    if error != nil {
        log.Println("Could not select welcome screen from directory", welcomepath, ":", error)
        return
    }

    fileinfos := list.New()
    for _, fi := range allFileinfos {
        fileinfos.PushBack(fi)
    }

    for {
        numFileinfos := fileinfos.Len()
        if numFileinfos == 0 {
            log.Println("Found no readable welcome screens in directory", welcomepath)
            break
        }

        selection := rand.Intn(numFileinfos)
        e := fileinfos.Front()
        for i := 0; e != nil && i < selection; i++ {
            e = e.Next()
        }
        selectedinfo := e.Value.(os.FileInfo)

        selectedpath := path.Join(welcomepath, selectedinfo.Name())
        welcomescreen, error := ioutil.ReadFile(selectedpath)
        if error == nil {
            // Loaded fine! We're done!
            log.Println("Loaded new welcome screen from", selectedpath)
            screen.screenText = string(welcomescreen)
            return
        }

        // Didn't load, so don't try that one again.
        log.Println("Couldn't load welcome screen", selectedpath, "(so skipping it) due to error:", error)
        fileinfos.Remove(e)
    }

    log.Println("Oops, ran out of welcome screens to try to load, keeping the old one")
}

func (screen *welcomeScreen) provideScreens() {
    for {
        now := time.Now()

        if now.After(screen.loadAfter) {
            log.Println("Time", now, "is after", screen.loadAfter, "so loading a new screen")
            screen.chooseScreen()

            _, _, secs := now.Clock()
            nowSeconds := time.Duration(-secs) * time.Second
            oneMinute := time.Duration(1) * time.Minute
            screen.loadAfter = now.Add(nowSeconds + oneMinute)
        }

        screen.Screens <- screen.screenText
        log.Println("Gave that channel a welcome screen (channels love welcome screens)")
    }
}

func (screen *welcomeScreen) Welcome(client *Client) {
    screenText := <-screen.Screens
    client.ToClient <- screenText

    INPUT: for input := range client.ToServer {
        parts := strings.SplitN(input, " ", 2)
        command := strings.ToLower(parts[0])

        if strings.HasPrefix("connect", command) {
            if len(parts) > 1 {
                parts = strings.SplitN(parts[1], " ", 2)
            }
            if len(parts) < 2 {
                client.ToClient <- "An account name and password are required to connect. Try 'connect <name> <password>'."
                continue INPUT
            }

            accountName, password := parts[0], parts[1]
            account, error := VerifyAccount(accountName, password)
            if error != nil {
                client.ToClient <- "Couldn't log you in as " + accountName + ": " + error.Error()
                continue INPUT
            }

            go Game(client, account)
            break INPUT

        } else if strings.HasPrefix("register", command) {

            if len(parts) > 1 {
                parts = strings.SplitN(parts[1], " ", 2)
            }
            if len(parts) < 2 {
                client.ToClient <- "An account name and password are required to connect. Try 'register <name> <password>'."
                continue INPUT
            }

            accountName, password := parts[0], parts[1]
            account, error := RegisterAccount(accountName, password)
            if error != nil {
                client.ToClient <- "Couldn't register you as " + accountName + ": " + error.Error()
                continue INPUT
            }

            go Game(client, account)
            break INPUT

        } else if command == "derp" {
            client.ToClient <- "DERP INDEED"
        } else if command == "herp" {
            client.ToClient <- "~herp~"
        } else {
            screenText := <-screen.Screens
            client.ToClient <- screenText
        }
    }
}
