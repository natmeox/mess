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
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type key int

const (
	ContextKeyAccount key = iota
	ContextKeyThing
)

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
	var paletteItems []*Thing
	for i := 0; i < 10; i++ {
		thing := World.ThingForId(ThingId(i))
		if thing != nil {
			paletteItems = append(paletteItems, thing)
		}
	}

	context := map[string]interface{}{
		"CsrfToken": nosurf.Token(r),
		"Config": map[string]interface{}{
			"Debug":       Config.Debug,
			"ServiceName": Config.ServiceName,
			"HostName":    Config.HostName,
		},
		"Account":      context.Get(r, ContextKeyAccount), // could be nil
		"PaletteItems": paletteItems,
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

var webThingMux *http.ServeMux

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

func WebThingTable(w http.ResponseWriter, r *http.Request) {
	thing := context.Get(r, ContextKeyThing).(*Thing)
	account := context.Get(r, ContextKeyAccount).(*Account)

	if !thing.EditableById(account.Character) {
		http.Error(w, "No access to table data", http.StatusForbidden)
		return
	}

	if r.Method == "POST" {
		updateText := r.PostFormValue("updated_data")
		var updates map[string]interface{}
		err := json.Unmarshal([]byte(updateText), &updates)
		if err != nil {
			// aw carp
			// TODO: set a flash?
			http.Redirect(w, r, fmt.Sprintf("%stable", thing.GetURL()), http.StatusSeeOther)
			return
		}

		deleteText := r.PostFormValue("deleted_data")
		var deletes map[string]interface{}
		err = json.Unmarshal([]byte(deleteText), &deletes)
		if err != nil {
			// aw carp
			// TODO: set a flash?
			http.Redirect(w, r, fmt.Sprintf("%stable", thing.GetURL()), http.StatusSeeOther)
			return
		}

		thing.Table = mergeMapInto(updates, thing.Table)
		thing.Table = deleteMapFrom(deletes, thing.Table)
		World.SaveThing(thing)

		http.Redirect(w, r, fmt.Sprintf("%stable", thing.GetURL()), http.StatusSeeOther)
		return
	}

	RenderTemplate(w, r, "thing/page/table.html", map[string]interface{}{
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

func WebThingProgram(w http.ResponseWriter, r *http.Request) {
	thing := context.Get(r, ContextKeyThing).(*Thing)
	account := context.Get(r, ContextKeyAccount).(*Account)

	if !thing.EditableById(account.Character) {
		http.Error(w, "No access to program", http.StatusForbidden)
		return
	}

	if r.Method == "POST" {
		// TODO: try compiling the text first?
		program := r.PostFormValue("text")

		thing.Program = NewProgram(program)
		World.SaveThing(thing)
	}

	RenderTemplate(w, r, "thing/page/program.html", map[string]interface{}{
		"Title": fmt.Sprintf("Edit program – %s", thing.Name),
		"Thing": thing,
	})
}

func WebThingAccess(w http.ResponseWriter, r *http.Request) {
	account := context.Get(r, ContextKeyAccount).(*Account)
	thing := context.Get(r, ContextKeyThing).(*Thing)

	// Only the owner can edit the access lists.
	if !thing.OwnedById(account.Character) {
		http.Error(w, "No access to access lists", http.StatusForbidden)
		return
	}

	if r.Method == "POST" {
		changed := false

		if adminsText := r.PostFormValue("admins"); adminsText != "" {
			var adminIds []ThingId
			err := json.Unmarshal([]byte(adminsText), &adminIds)
			if err != nil {
				// TODO: set a flash
				http.Redirect(w, r, fmt.Sprintf("%saccess", thing.GetURL()), http.StatusSeeOther)
				return
			}

			thing.AdminList = adminIds
			changed = true
		}

		// TODO: handle allows

		if deniedText := r.PostFormValue("denied"); deniedText != "" {
			var deniedIds []ThingId
			err := json.Unmarshal([]byte(deniedText), &deniedIds)
			if err != nil {
				// TODO: set a flash
				http.Redirect(w, r, fmt.Sprintf("%saccess", thing.GetURL()), http.StatusSeeOther)
				return
			}

			thing.DenyList = deniedIds
			changed = true
		}

		if changed {
			World.SaveThing(thing)
		}

		http.Redirect(w, r, fmt.Sprintf("%saccess", thing.GetURL()), http.StatusSeeOther)
		return
	}

	RenderTemplate(w, r, "thing/page/access.html", map[string]interface{}{
		"Title": thing.Name,
		"Thing": thing,
	})
}

func WebThingEdit(w http.ResponseWriter, r *http.Request) {
	thing := context.Get(r, ContextKeyThing).(*Thing)
	account := context.Get(r, ContextKeyAccount).(*Account)

	if !thing.EditableById(account.Character) {
		RenderTemplate(w, r, "thing/thing-no-edit.html", map[string]interface{}{
			"Title": thing.Name,
			"Thing": thing,
		})
		return
	}

	if r.Method == "POST" {
		// TODO: validate??
		thing.Table["description"] = r.PostFormValue("description")
		if thing.Type == PlayerThing {
			thing.Table["glance"] = r.PostFormValue("glance")
			thing.Table["pronouns"] = r.PostFormValue("pronouns")
		}

		parentIdStr := r.PostFormValue("parent")
		parentId64, err := strconv.ParseInt(parentIdStr, 10, 64)
		if err != nil {
			// TODO: set a flash? cause an error? eh
		} else {
			parentId := ThingId(parentId64)
			newParent := World.ThingForId(parentId)
			// TODO: does the viewer control newParent sufficiently to move the thing there?
			if newParent == nil {
				// TODO: set a flash? cause an error? eh
			} else {
				thing.Parent = parentId
			}
		}

		World.SaveThing(thing)

		http.Redirect(w, r, thing.GetURL(), http.StatusSeeOther)
		return
	}

	typeName := thing.Type.String()
	templateName := fmt.Sprintf("thing/type/%s.html", typeName)
	RenderTemplate(w, r, templateName, map[string]interface{}{
		"Title": fmt.Sprintf("Edit access lists - %s", thing.Name),
		"Thing": thing,
	})
}

func WebThing(w http.ResponseWriter, r *http.Request) {
	// path: <0>/<1: type name>/<2: thing id>/<3+: further page arguments>
	pathParts := strings.Split(r.URL.Path, "/")
	thingIdStr := pathParts[2]
	thingId, err := strconv.ParseInt(thingIdStr, 10, 64)
	if err != nil {
		log.Println("Error converting /thing/ argument", thingIdStr, "to number:", err.Error())
		http.NotFound(w, r)
		return
	}
	thing := World.ThingForId(ThingId(thingId))
	if thing == nil {
		// regular ol' expected not-found this time
		http.NotFound(w, r)
		return
	}

	// Make sure we're at the right URL.
	pathType := pathParts[1]
	typeName := thing.Type.String()
	if len(pathParts) < 4 || pathType != typeName {
		newPath := fmt.Sprintf("/%s/%d/", typeName, thingId)
		http.Redirect(w, r, newPath, http.StatusMovedPermanently)
		return
	}

	context.Set(r, ContextKeyThing, thing)

	r.URL.Path = fmt.Sprintf("/%s", strings.Join(pathParts[3:], "/"))
	webThingMux.ServeHTTP(w, r)
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
		tmpls := template.New("")
		err := filepath.Walk("./template", func (path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".html") {
				return err
			}

			relativePath, err := filepath.Rel("./template", path)
			if err != nil {
				return err
			}

			subtmpl, err := template.ParseFiles(path)
			if err != nil {
				return err
			}

			_, err = tmpls.AddParseTree(relativePath, subtmpl.Tree)
			return err
		})
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

	webThingHandler := RequireAccountFunc(WebThing)
	http.Handle("/thing/", webThingHandler)
	http.Handle("/player/", webThingHandler)
	http.Handle("/place/", webThingHandler)
	http.Handle("/action/", webThingHandler)
	http.Handle("/program/", webThingHandler)

	webThingMux = http.NewServeMux()
	webThingMux.HandleFunc("/", WebThingEdit)
	webThingMux.HandleFunc("/table", WebThingTable)
	webThingMux.HandleFunc("/program", WebThingProgram)
	webThingMux.HandleFunc("/access", WebThingAccess)

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
