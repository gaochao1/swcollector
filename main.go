package main

import (
	"flag"
	"fmt"
	"os"

	_ "github.com/baishancloud/goperfcounter"
	"github.com/baishancloud/octopux-swcollector/cron"
	"github.com/baishancloud/octopux-swcollector/funcs"
	"github.com/baishancloud/octopux-swcollector/g"
	"github.com/baishancloud/octopux-swcollector/http"
	"github.com/getsentry/raven-go"
)

func init() {
	//raven.SetDSN("")
	raven.SetDSN("testdsn")
}

func main() {

	cfg := flag.String("c", "cfg.json", "configuration file")
	version := flag.Bool("v", false, "show version")

	flag.Parse()

	if *version {
		fmt.Println(g.VERSION)
		os.Exit(0)
	}

	g.ParseConfig(*cfg)

	g.GetHost()
	g.InitLocalIps()
	g.InitFaceIp()

	funcs.BuildMappers()
	cron.Collect()

	go http.Start()

	select {}

}
