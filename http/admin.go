package http

import (
	"net/http"

	"github.com/toolkits/file"
)

func configAdminRoutes() {

	http.HandleFunc("/workdir", func(w http.ResponseWriter, r *http.Request) {
		RenderDataJson(w, file.SelfDir())
	})

}
