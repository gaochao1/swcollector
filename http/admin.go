package http

import (
	"net/http"
	"os"
	"time"

	"github.com/gaochao1/swcollector/g"

	"github.com/toolkits/file"
)

func configAdminRoutes() {

	http.HandleFunc("/workdir", func(w http.ResponseWriter, r *http.Request) {
		RenderDataJson(w, file.SelfDir())
	})
	http.HandleFunc("/ips", func(w http.ResponseWriter, r *http.Request) {
		RenderDataJson(w, g.TrustableIps())
	})
	http.HandleFunc("/exit", func(w http.ResponseWriter, r *http.Request) {
		if g.IsTrustable(r.RemoteAddr) {
			w.Write([]byte("exiting..."))
			go func() {
				time.Sleep(time.Second)
				os.Exit(0)
			}()
		} else {
			w.Write([]byte("no privilege"))
		}
	})
	http.HandleFunc("/config/reload", func(w http.ResponseWriter, r *http.Request) {
		if g.IsTrustable(r.RemoteAddr) {
			g.ParseConfig(g.ConfigFile)
			if g.Config().SwitchHosts.Enabled {
				hostcfg := g.Config().SwitchHosts.Hosts
				g.ParseHostConfig(hostcfg)
			}
			if g.Config().CustomMetrics.Enabled {
				custMetrics := g.Config().CustomMetrics.Template
				g.ParseCustConfig(custMetrics)
			}
			RenderDataJson(w, g.Config())
		} else {
			w.Write([]byte("no privilege"))
		}
	})
}
