package funcs

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gaochao1/swcollector/g"
)

type EcmcRes struct {
	Dat json.RawMessage `json:"dat"`
	Err string          `json:"err"`
}

type Host struct {
	ID     int64  `json:"id"`
	SN     string `json:"sn"`
	IP     string `json:"ip"`
	Name   string `json:"name"`
	Note   string `json:"note"`
	Cpu    string `json:"cpu"`
	Mem    string `json:"mem"`
	Disk   string `json:"disk"`
	Cate   string `json:"cate"`
	Tenant string `json:"tenant"`
}

type n9eHost struct {
	ID    int64  `json:"id"`
	Ident string `json:"ident"`
	Alias string `json:"alias"`
}

type NodeHosts struct {
	List  []Host `json:"list"`
	Total int64  `json:"total"`
}

// HTTPGet 发起一个 http get 请求
func HTTPGet(url string, headers map[string]string) (body []byte, err error) {
	body = []byte{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   15 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ = ioutil.ReadAll(resp.Body)
		erroMsg := fmt.Sprintf("HTTP Connect Failed, Code is %d, body is %s", resp.StatusCode, string(body))
		err = errors.New(erroMsg)
		return
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	return
}

func GetNodeHosts(nodeID int64) (hosts []Host, err error) {
	p := 1
	for {
		apiAddr := fmt.Sprintf("%s/api/hsp/node/obj/%d/host?p=%d", g.Config().Ecmc.Addr, nodeID, p)
		headers := map[string]string{}
		headers["x-user-token"] = g.Config().Ecmc.Token
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
		var nodeHosts NodeHosts
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

func GetAllByIpByEcmc() (ips []string) {
	ipMap := map[string]bool{}
	for _, id := range g.Config().Ecmc.Nodes {
		hosts, err := GetNodeHosts(id)
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
