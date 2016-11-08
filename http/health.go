package http

import (
	"net/http"

	"github.com/baishancloud/octopux-swcollector/funcs/lansw"
	"github.com/baishancloud/octopux-swcollector/g"
)

func configHealthRoutes() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(g.VERSION))
	})

	http.HandleFunc("/iscollector", func(w http.ResponseWriter, r *http.Request) {
		if lansw.IsCollector {
			w.Write([]byte("true"))
		} else {
			w.WriteHeader(400)
			//w.Write([]byte("false"))
		}

	})
}
