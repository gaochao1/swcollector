package funcs

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"log"

	"github.com/gaochao1/swcollector/g"
)

type N9eNodeHosts struct {
	List  []N9eHost `json:"list"`
	Total int64     `json:"total"`
}

type N9eHost struct {
	ID    int64  `json:"id"`
	IP    string `json:"ident"`
	Alias string `json:"alias"`
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func GetN9eHosts(nodeID int64) (hosts []N9eHost, err error) {
	p := 1
	for {
		apiAddr := fmt.Sprintf("%s/api/portal/node/%d/endpoint?p=%d", g.Config().N9e.Addr, nodeID, p)
		headers := map[string]string{}
		headers["Authorization"] = "Basic " + basicAuth(g.Config().N9e.User, g.Config().N9e.Pass)
		var res []byte
		res, err = HTTPGet(apiAddr, headers)
		if err != nil {
			return
		}
		var ecmcRes EcmcRes
		if err = json.Unmarshal(res, &ecmcRes); err != nil {
			return
		}
		if ecmcRes.Err != "" {
			err = errors.New(ecmcRes.Err)
			return
		}
		var nodeHosts N9eNodeHosts
		if err = json.Unmarshal(ecmcRes.Dat, &nodeHosts); err != nil {
			return
		}
		if len(nodeHosts.List) == 0 {
			break
		}
		hosts = append(hosts, nodeHosts.List...)
		p = p + 1
	}
	return
}

func GetAllByIpByN9e() (ips []string) {
	ipMap := map[string]bool{}
	for _, id := range g.Config().N9e.Nodes {
		hosts, err := GetN9eHosts(id)
		if err != nil {
			log.Println(err)
			continue
		}
		for _, host := range hosts {
			if _, ok := ipMap[host.IP]; ok {
				continue
			}
			ipMap[host.IP] = true
			ips = append(ips, host.IP)
		}
	}
	return
}
