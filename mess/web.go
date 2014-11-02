package mess

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/justinas/nosurf"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type key int

const ContextKeyAccount key = 0

var store *sessions.CookieStore
var getTemplate func(string) *template.Template

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

func RenderTemplate(w http.ResponseWriter, r *http.Request, templateName string, templateContext map[string]interface{}) {
	context := map[string]interface{}{
		"CsrfToken": nosurf.Token(r),
		"Config": map[string]interface{}{
			"Debug":       Config.Debug,
			"ServiceName": Config.ServiceName,
			"HostName":    Config.HostName,
		},
		"Account": context.Get(r, ContextKeyAccount), // could be nil
	}
	// If e.g. Account was provided by the caller, it overrides our default one.
	for k, v := range templateContext {
		context[k] = v
	}

	template := getTemplate(templateName)
	err := template.Execute(w, context)
	if err != nil {
		log.Println("Error executing index.html template:", err.Error())
	}
}

func RequireAccount(h http.Handler) http.Handler {
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

		context.Set(r, ContextKeyAccount, acc)
		h.ServeHTTP(w, r)
	})
}

func RequireAccountFunc(f func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return RequireAccount(http.HandlerFunc(f))
}

func WebSignIn(w http.ResponseWriter, r *http.Request) {
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

	RenderTemplate(w, r, "signin.html", map[string]interface{}{
		"CsrfToken": nosurf.Token(r),
		"Title":     "Sign in",
	})
}

func WebSignOut(w http.ResponseWriter, r *http.Request) {
	// Don't really care if there's an account already or no.
	SetAccountForRequest(w, r, nil)
	RenderTemplate(w, r, "signout.html", map[string]interface{}{
		"Title": "Sign out",
	})
}

func mergeMapInto(source map[string]interface{}, target map[string]interface{}) map[string]interface{} {
	for key, value := range source {
		switch value.(type) {
		case map[string]interface{}:
			valueMap := value.(map[string]interface{})
			nextTarget := target[key]
			switch nextTarget.(type) {
			case map[string]interface{}:
				target[key] = mergeMapInto(valueMap, nextTarget.(map[string]interface{}))
			default:
				// Well... we are replacing target values with source ones. So this type mismatch is allowed.
				target[key] = valueMap
			}
		default:
			target[key] = value
		}
	}
	return target
}

func deleteMapFrom(source map[string]interface{}, target map[string]interface{}) map[string]interface{} {
	for key, value := range source {
		switch value.(type) {
		case map[string]interface{}:
			valueMap := value.(map[string]interface{})
			nextTarget := target[key]
			switch nextTarget.(type) {
			case map[string]interface{}:
				target[key] = deleteMapFrom(valueMap, nextTarget.(map[string]interface{}))
			default:
				// Uhhh... if the target is no longer a map, then any further deleted keys from deeper in the source are "deleted". So yay?
				break
			}
		default:
			delete(target, key)
		}
	}
	return target
}

func WebTable(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	thingIdStr := pathParts[2]
	thingId, err := strconv.ParseInt(thingIdStr, 10, 64)
	if err != nil {
		log.Println("Error converting /thing/ argument", thingIdStr, "to number:", err.Error())
		http.NotFound(w, r)
		return
	}
	thing := World.ThingForId(int(thingId))
	if thing == nil {
		// regular ol' expected not-found this time
		http.NotFound(w, r)
		return
	}

	// TODO: permit only some editing once there are permissions

	if r.Method == "POST" {
		updateText := r.PostFormValue("updated_data")
		var updates map[string]interface{}
		err := json.Unmarshal([]byte(updateText), &updates)
		if err != nil {
			// aw carp
			// TODO: set a flash?
			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
			return
		}

		deleteText := r.PostFormValue("deleted_data")
		var deletes map[string]interface{}
		err = json.Unmarshal([]byte(deleteText), &deletes)
		if err != nil {
			// aw carp
			// TODO: set a flash?
			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
			return
		}

		thing.Table = mergeMapInto(updates, thing.Table)
		thing.Table = deleteMapFrom(deletes, thing.Table)
		World.SaveThing(thing)

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
		return
	}

	RenderTemplate(w, r, "table.html", map[string]interface{}{
		"Title": fmt.Sprintf("Edit all data – %s", thing.Name),
		"Thing": thing,
		"json": func(v interface{}) interface{} {
			output, err := json.MarshalIndent(v, "", "    ")
			if err != nil {
				escapedError := template.JSEscapeString(err.Error())
				message := fmt.Sprintf("/* error encoding JSON: %s */ {}", escapedError)
				return template.JS(message)
			}
			return template.JS(output)
		},
	})
}

func WebThing(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	thingIdStr := pathParts[2]
	thingId, err := strconv.ParseInt(thingIdStr, 10, 64)
	if err != nil {
		log.Println("Error converting /thing/ argument", thingIdStr, "to number:", err.Error())
		http.NotFound(w, r)
		return
	}
	thing := World.ThingForId(int(thingId))
	if thing == nil {
		// regular ol' expected not-found this time
		http.NotFound(w, r)
		return
	}

	// TODO: permit only some editing once there are permissions

	if r.Method == "POST" {
		// TODO: validate??
		thing.Table["glance"] = r.PostFormValue("glance")
		thing.Table["description"] = r.PostFormValue("description")
		thing.Table["pronouns"] = r.PostFormValue("pronouns")
		World.SaveThing(thing)

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
		return
	}

	RenderTemplate(w, r, "thing.html", map[string]interface{}{
		"Title": thing.Name,
		"Thing": thing,
	})
}

func WebIndex(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, r, "index.html", map[string]interface{}{
		"Title": "Home",
	})
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
	if Config.Debug {
		getTemplate = func(name string) *template.Template {
			templates = ParseTemplates()
			return LookupTemplate(name)
		}
	} else {
		getTemplate = LookupTemplate
	}

	http.HandleFunc("/signin", WebSignIn)
	http.HandleFunc("/signout", WebSignOut)
	http.Handle("/table/", RequireAccountFunc(WebTable))
	http.Handle("/thing/", RequireAccountFunc(WebThing))

	indexHandler := RequireAccountFunc(WebIndex)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() != "/" {
			http.NotFound(w, r)
			return
		}
		indexHandler.ServeHTTP(w, r)
	})

	log.Println("Listening for web requests at address", Config.WebAddress)
	webHandler := context.ClearHandler(nosurf.New(http.DefaultServeMux))
	http.ListenAndServe(Config.WebAddress, webHandler)
}
