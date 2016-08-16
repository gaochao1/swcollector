package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gaochao1/swcollector/cron"
	"github.com/gaochao1/swcollector/funcs"
	"github.com/gaochao1/swcollector/g"
	"github.com/gaochao1/swcollector/http"
)

func main() {

	cfg := flag.String("c", "cfg.json", "configuration file")
	version := flag.Bool("v", false, "show version")
	check := flag.Bool("check", false, "check collector")

	flag.Parse()

	if *version {
		fmt.Println(g.VERSION)
		os.Exit(0)
	}

	g.ParseConfig(*cfg)

	g.InitRootDir()
	g.InitLocalIps()
	g.InitRpcClients()

	if *check {
		funcs.CheckCollector()
		os.Exit(0)
	}

	funcs.BuildMappers()

	cron.Collect()

	go http.Start()

	select {}

}
