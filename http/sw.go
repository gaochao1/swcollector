package http

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/baishancloud/swcollector/funcs"
	"github.com/baishancloud/swcollector/g"
)

func configSwRoutes() {

	http.HandleFunc("/page/sw/time", func(w http.ResponseWriter, req *http.Request) {
		RenderDataJson(w, time.Now().Format("2006-01-02 15:04:05"))
	})

	http.HandleFunc("/page/sw/iprange", func(w http.ResponseWriter, req *http.Request) {
		RenderDataJson(w, strings.Join(g.Config().Switch.IpRange, "\n"))
	})

	http.HandleFunc("/page/sw/live", func(w http.ResponseWriter, req *http.Request) {
		RenderDataJson(w, len(funcs.AliveIp))
	})

	http.HandleFunc("/page/sw/list", func(w http.ResponseWriter, r *http.Request) {

		var ret [][]interface{} = make([][]interface{}, 0)
		for _, swSystem := range funcs.SwSystemInfo() {
			ret = append(ret,
				[]interface{}{
					swSystem.Ip,
					swSystem.Hostname,
					swSystem.Model,
					swSystem.Uptime,
					fmt.Sprintf("%d%%", swSystem.Cpu),
					fmt.Sprintf("%d%%", swSystem.Mem),
					fmt.Sprintf("%sms", swSystem.Ping),
				})
		}
		RenderDataJson(w, ret)
	})
}
