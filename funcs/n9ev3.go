package funcs

import (
	"encoding/json"
	"errors"
	"fmt"

	"log"

	"github.com/gaochao1/swcollector/g"
)

type N9eV3NodeHosts struct {
	List  []N9eV3Host `json:"list"`
	Total int64       `json:"total"`
}

type N9eV3Host struct {
	ID          int64  `json:"id"`
	UUID        string `json:"uuid"`
	Ident       string `json:"ident"`
	Name        string `json:"name"`
	Labels      string `json:"labels"`
	Note        string `json:"note"`
	Extend      string `json:"extend"`
	Cate        string `json:"cate"`
	Tenant      string `json:"tenant"`
	LastUpdated string `json:"last_updated"`
}

func GetN9eV3NodeHosts(nodeID int64) (hosts []N9eV3Host, err error) {
	p := 1
	for {
		apiAddr := fmt.Sprintf("%s/api/rdb/node/%d/resources?p=%d", g.Config().N9eV3.Addr, nodeID, p)
		headers := map[string]string{}
		headers["x-user-token"] = g.Config().N9eV3.Token
		var res []byte
		res, err = HTTPGet(apiAddr, headers)
		if err != nil {
			return
		}
		var n9eV3Res EcmcRes
		if err = json.Unmarshal(res, &n9eV3Res); err != nil {
			return
		}
		if n9eV3Res.Err != "" {
			err = errors.New(n9eV3Res.Err)
			return
		}
		var nodeHosts N9eV3NodeHosts
		if err = json.Unmarshal(n9eV3Res.Dat, &nodeHosts); err != nil {
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

func GetAllByIpByN9eV3() (ips []string) {
	ipMap := map[string]bool{}
	for _, id := range g.Config().N9eV3.Nodes {
		hosts, err := GetN9eV3NodeHosts(id)
		if err != nil {
			log.Println(err)
			continue
		}
		for _, host := range hosts {
			if _, ok := ipMap[host.Ident]; ok {
				continue
			}
			ipMap[host.Ident] = true
			ips = append(ips, host.Ident)
		}
	}
	return
}
