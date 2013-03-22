package mess

import (
	"fmt"
	"log"
	"net/http"
)

func StartWeb() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() != "/" {
			http.NotFound(w, r)
			return
		}

		fmt.Fprintf(w, "hi")
	})

	log.Println("Listening for web requests at address", Config.WebAddress)
	http.ListenAndServe(Config.WebAddress, nil)
}
