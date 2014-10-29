package mess

import (
	"github.com/gorilla/sessions"
	"html/template"
	"log"
	"net/http"
)

var store *sessions.CookieStore

func AccountForRequest(w http.ResponseWriter, r *http.Request) *Account {
	session, _ := store.Get(r, "session")
	accountNameValue, ok := session.Values["name"]
	if !ok {
		return nil
	}
	accountName, ok := accountNameValue.(string)
	if !ok {
		return nil
	}

	return GetAccount(accountName)
}

func StartWeb() {
	store = sessions.NewCookieStore([]byte(Config.CookieSecret))

	staticServer := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", staticServer))

	indexTemplate, err := template.ParseFiles("./template/index.html", "template/head.html", "template/foot.html")
	if err != nil {
		log.Fatalln("Couldn't load HTML templates:", err.Error())
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		acc := AccountForRequest(w, r) // might be nil

		if r.URL.String() != "/" {
			http.NotFound(w, r)
			return
		}

		err := indexTemplate.Execute(w, map[string]interface{}{"account": acc})
		if err != nil {
			log.Println("Error executing index.html template:", err.Error())
		}
	})

	log.Println("Listening for web requests at address", Config.WebAddress)
	http.ListenAndServe(Config.WebAddress, nil)
}
