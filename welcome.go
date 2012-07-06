package main

import (
    "container/list"
    "io/ioutil"
    "log"
    "math/rand"
    "os"
    "path"
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

        // TODO: we're assuming this blocks until someone wants to read one.
        screen.Screens <- screen.screenText
        log.Println("Gave that channel a welcome screen (channels love welcome screens)")
    }
}

func (screen *welcomeScreen) Welcome(client *Client) {
    screenText := <-screen.Screens
    client.ToClient <- screenText

    for command := range client.ToServer {
        if command == "derp" {
            client.ToClient <- "DERP INDEED"
        } else if command == "herp" {
            client.ToClient <- "~herp~"
        } else {
            screenText := <-screen.Screens
            client.ToClient <- screenText
        }
    }
}
