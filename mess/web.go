package mess

import (
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/justinas/nosurf"
	"html/template"
	"log"
	"net/http"
	"net/url"
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

	return Accounts.GetAccount(accountName)
}

func SetAccountForRequest(w http.ResponseWriter, r *http.Request, acc *Account) {
	session, _ := store.Get(r, "session")
	if acc != nil {
		session.Values["name"] = acc.LoginName
	} else {
		delete(session.Values, "name")
	}
	session.Save(r, w)
}

func StartWeb() {
	store = sessions.NewCookieStore([]byte(Config.CookieSecret))

	staticServer := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", staticServer))

	var templates *template.Template
	ParseTemplates := func() *template.Template {
		tmpls, err := template.ParseGlob("./template/*.html")
		if err != nil {
			log.Fatalln("Couldn't load HTML templates:", err.Error())
		}
		return tmpls
	}
	LookupTemplate := func(name string) *template.Template {
		tmpl := templates.Lookup(name)
		if tmpl == nil {
			log.Fatalln("Couldn't find HTML template", name)
		}
		return tmpl
	}

	// If in Debug mode, re-parse templates with each request.
	var GetTemplate func(string) *template.Template
	if Config.Debug {
		GetTemplate = func(name string) *template.Template {
			templates = ParseTemplates()
			return LookupTemplate(name)
		}
	} else {
		GetTemplate = LookupTemplate
	}

	RequireAccount := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			acc := AccountForRequest(w, r)
			if acc == nil {
				v := url.Values{}
				v.Set("next", r.URL.RequestURI())
				signinUrl := url.URL{
					Path:     "/signin",
					RawQuery: v.Encode(),
				}

				http.Redirect(w, r, signinUrl.RequestURI(), http.StatusTemporaryRedirect)
				return
			}

			h.ServeHTTP(w, r)
		})
	}

	http.HandleFunc("/signin", func(w http.ResponseWriter, r *http.Request) {
		acc := AccountForRequest(w, r)
		if acc != nil {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}

		if r.Method == "POST" {
			loginname := r.PostFormValue("name")
			password := r.PostFormValue("password")

			acc = Accounts.AccountForLogin(loginname, password)
			if acc != nil {
				SetAccountForRequest(w, r, acc)

				nextUrl := r.FormValue("next")
				if nextUrl == "" {
					nextUrl = "/"
				}
				http.Redirect(w, r, nextUrl, http.StatusTemporaryRedirect)
				return
			}
		}

		signinTemplate := GetTemplate("signin.html")
		err := signinTemplate.Execute(w, map[string]interface{}{
			"CsrfToken": nosurf.Token(r),
			"Title":     "Sign in",
		})
		if err != nil {
			log.Println("Error executing signin.html template:", err.Error())
		}
	})

	http.HandleFunc("/signout", func(w http.ResponseWriter, r *http.Request) {
		// Don't really care if there's an account already or no.
		SetAccountForRequest(w, r, nil)

		signoutTemplate := GetTemplate("signout.html")
		err := signoutTemplate.Execute(w, map[string]interface{}{
			"Title": "Sign out",
		})
		if err != nil {
			log.Println("Error executing signout.html template:", err.Error())
		}
	})

	http.Handle("/", RequireAccount(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() != "/" {
			http.NotFound(w, r)
			return
		}

		acc := AccountForRequest(w, r) // might be nil

		indexTemplate := GetTemplate("index.html")
		err := indexTemplate.Execute(w, map[string]interface{}{
			"Title":   "Hi",
			"Account": acc,
		})
		if err != nil {
			log.Println("Error executing index.html template:", err.Error())
		}
	})))

	log.Println("Listening for web requests at address", Config.WebAddress)
	webHandler := context.ClearHandler(nosurf.New(http.DefaultServeMux))
	http.ListenAndServe(Config.WebAddress, webHandler)
}
