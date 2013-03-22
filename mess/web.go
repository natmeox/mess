package mess

import (
	"html/template"
	"log"
	"net/http"
)

func StartWeb() {
	staticServer := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", staticServer))

	indexTemplate, err := template.ParseFiles("./template/index.html", "template/head.html", "template/foot.html")
	if err != nil {
		log.Fatalln("Couldn't load HTML templates:", err.Error())
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() != "/" {
			http.NotFound(w, r)
			return
		}

		err := indexTemplate.Execute(w, nil)
		if err != nil {
			log.Println("Error executing index.html template:", err.Error())
		}
	})

	log.Println("Listening for web requests at address", Config.WebAddress)
	http.ListenAndServe(Config.WebAddress, nil)
}
