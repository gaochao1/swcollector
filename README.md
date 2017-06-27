

基于小米运维开源的[open-falcon](http://open-falcon.com)，交换机专用agent。

感谢小米运维同学的杰出工作、开源、良好的文档和热心。

感谢[来炜](https://github.com/laiwei)的宝贵建议。

##简介
采集的metric列表：

* CPU利用率
* 内存利用率
* Ping延时（正常返回延时，超时返回 -1，可以用于存活告警）
* 网络连接数（监控防火墙时，目前仅支持Cisco ASA)
* IfHCInOctets
* IfHCOutOctets
* IfHCInUcastPkts
* IfHCOutUcastPkts
* IfHCInBroadcastPkts
* IfHCOutBroadcastPkts
* IfHCInMulticastPkts
* IfHCOutMulticastPkts
* IfInDiscards
* IfOutDiscards
* IfInErrors
* IfOutErros
* IfInUnknownProtos
* IfOutQLen
* IfSpeed
* IfSpeedPercent 
* IfOperStatus(接口状态，1 up, 2 down, 3 testing, 4 unknown, 5 dormant, 6 notPresent, 7 lowerLayerDown)
	

CPU和内存的OID私有，根据设备厂家和OS版本可能不同。目前测试过的设备：

* Cisco IOS(Version 12)
* Cisco NX-OS(Version 6)
* Cisco IOS XR(Version 5)
* Cisco IOS XE(Version 15)
* Cisco ASA (Version 9)
* Ruijie 10G Routing Switch
* Huawei VRP(Version 8)
* Huawei VRP(Version 5.20)
* Huawei VRP(Version 5.120)
* Huawei VRP(Version 5.130)
* Huawei VRP(Version 5.70)
* Juniper JUNOS(Version 10)
* H3C(Version 5)
* H3C(Version 5.20)
* H3C(Version 7)

##二进制安装
从[这里](https://github.com/gaochao1/swcollector/releases) 下载编译好的最新二进制版本即可。注意：这些二进制只能跑在64位Linux上

##源码安装
```
	依赖$GOPATH/src/github.com/gaochao1/sw
	cd $GOPATH/src/github.com/gaochao1/swcollector
	go get ./...
	chmod +x control
	./control build
	./control pack
	最后一步会pack出一个tar.gz的安装包，拿着这个包去部署服务即可。
	
	升级时，确保先更新sw
	cd $GOPATH/src/github.com/gaochao1/sw
	git pull
```

##部署说明

swcollector需要部署到有交换机SNMP访问权限的服务器上。

使用Go原生的ICMP协议进行Ping探测，swcollector需要root权限运行。

支持使用 Gosnmp 或 snmpwalk 进行数据采集，如果使用 snmpwalk 模式，需要在监控探针服务器上安装个snmpwalk命令

##配置说明

配置文件请参照cfg.example.json，修改该文件名为cfg.json，将该文件里的IP换成实际使用的IP。

配置说明：
```
{
    "debug": true,
	"debugmetric":{                                          # 在日志中 debug 具体的指标
		"endpoints":["192.168.56.101","192.168.56.102"],     # 必填
		"metrics":["switch.if.In","switch.if.Out"],          # 必填
		"tags":"ifName=Fa0/1"  # 有则匹配 tag,如为 "" 则打印该 metric 的全部信息
	},
	"switch":{
	   "enabled": true,
		"ipRange":[            #交换机IP地址段，对该网段有效IP，先发Ping包探测，对存活IP发SNMP请求
            "192.168.56.101/32",      
            "192.168.56.102-192.168.56.120",#现在支持这样的配置方式，对区域内的ip进行ping探测，对存活ip发起snmp请求。
            "172.16.114.233" 
 		],
		"gosnmp":true,         #是否使用 gosnmp 采集, false 则使用 snmpwalk
 		"pingTimeout":300,     #Ping超时时间，单位毫秒
		"pingRetry":4,         #Ping探测重试次数
		"community":"public",  #SNMP认证字符串
		"snmpTimeout":1000,    #SNMP超时时间，单位毫秒
		"snmpRetry":5,         #SNMP重试次数
		"ignoreIface": ["Nu","NU","Vlan","Vl"], #忽略的接口，如Nu匹配ifName为*Nu*的接口
		"ignoreOperStatus": true,               #不采集IfOperStatus
		"speedlimit":0,  #流量的上限，如果采集计算出的流量超过这个数值，则抛弃不上报。如果填0，则以接口的速率（ifSpeed)作为上限计算。注意 interface vlan 这样的虚接口是没有 ifSpeed 的，因此不进行最大值的比较。
		"ignorePkt": true,                      #不采集IfHCInUcastPkts和IfHCOutUcastPkts
		"pktlimit": 0,  #pkt的上限，如果采集计算出的包速率超过这个数值，则抛弃不上报。如果填0，则不进行最大值比较。
		"ignoreBroadcastPkt": true,  #不采集IfHCInBroadcastPkts和IfHCOutBroadcastPkts
		"broadcastPktlimit": 0,  #broadcastPkt的上限，如果采集计算出的包速率超过这个数值，则抛弃不上报。如果填0，则不进行最大值比较。
		"ignoreMulticastPkt": true,  #不采集IfHCInMulticastPkts和IfHCOutMulticastPkts
		"multicastPktlimit": 0,  #multicastPkt的上限，如果采集计算出的包速率超过这个数值，则抛弃不上报。如果填0，则不进行最大值比较。
		"ignoreDiscards": true,   #不采集IfInDiscards和IfOutDiscards
		"discardsPktlimit": 0,   #discardsPkt的上限，如果采集计算出的包速率超过这个数值，则抛弃不上报。如果填0，则不进行最大值比较。
		"ignoreErrors": true,   #不采集IfInErrors和IfOutErros
		"errorsPktlimit": 0,    #errorsPkt的上限，如果采集计算出的包速率超过这个数值，则抛弃不上报。如果填0，则不进行最大值比较。
		"ignoreUnknownProtos":true,    #不采集IfInUnknownProtos
		"unknownProtosPktlimit": 0,    #unknownProtosPkt的上限，如果采集计算出的包速率超过这个数值，则抛弃不上报。如果填0，则不进行最大值比较。
		"ignoreOutQLen":true,    #不采集IfOutQLen
		"outQLenPktlimit": 0,   #outQLenPkt的上限，如果采集计算出的包速率超过这个数值，则抛弃不上报。如果填0，则不进行最大值比较。
		"fastPingMode": true,  
		"limitConcur": 1000, #交换机采集的并发限制
		"limitCon": 4 #对于单台交换机上，多个指标采集的并发限制
 	}, 
	"switchhosts":{
		"enabled":true,
		"hosts":"./hosts.json"  #自定义的host与Ip地址对应表，如果配置，则上报时会用这里的host替换ip地址
	},
	"customMetrics":{         
		"enabled":true,
		"template":"./custom.json"    #自定义的metric列表，如果配置，会根据这个配置文件，采集额外的自定义metric
	},
    "transfer": {
        "enabled": true,
        "addr": "127.0.0.1:8433",
        "interval": 60,
        "timeout": 1000
    },
    "http": {
        "enabled": true,
        "listen": ":1989"
    }
}

```

##部署说明
由于是并发采集，因此每个周期的采集耗时，事实上取决于被采集的交换机中，最慢的那个。
因此我们可以在 debug 模式下观察每个交换机的采集耗时。
```
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.10.1 PingResult: true len_list: 440 UsedTime: 5
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.10.252 PingResult: true len_list: 97 UsedTime: 2
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.13.1 PingResult: true len_list: 24 UsedTime: 1
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.14.1 PingResult: true len_list: 23 UsedTime: 1
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.23.1 PingResult: true len_list: 61 UsedTime: 2
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.12.1 PingResult: true len_list: 55 UsedTime: 1
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.12.5 PingResult: true len_list: 26 UsedTime: 2
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.12.6 PingResult: true len_list: 26 UsedTime: 2
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.12.11 PingResult: true len_list: 26 UsedTime: 2
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.12.12 PingResult: true len_list: 26 UsedTime: 2
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.12.13 PingResult: true len_list: 26 UsedTime: 2
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.12.14 PingResult: true len_list: 26 UsedTime: 2
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.12.15 PingResult: true len_list: 26 UsedTime: 2
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.12.16 PingResult: true len_list: 26 UsedTime: 2
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.12.17 PingResult: true len_list: 26 UsedTime: 2
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.12.18 PingResult: true len_list: 26 UsedTime: 2
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.12.19 PingResult: true len_list: 26 UsedTime: 2
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.12.20 PingResult: true len_list: 26 UsedTime: 2
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.12.21 PingResult: true len_list: 26 UsedTime: 2
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.12.22 PingResult: true len_list: 26 UsedTime: 2
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.12.23 PingResult: true len_list: 26 UsedTime: 2
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.12.24 PingResult: true len_list: 26 UsedTime: 2
2016/08/16 21:31:24 swifstat.go:121: IP: 192.168.12.25 PingResult: true len_list: 26 UsedTime: 2
2016/08/16 21:31:29 swifstat.go:121: IP: 192.168.11.2 PingResult: true len_list: 348 UsedTime: 10
2016/08/16 21:31:29 swifstat.go:177: UpdateIfStats complete. Process time 10.700895998s.
```
如下所示，我们可以发现 192.168.11.2 这个交换机的采集耗时最长，用掉了 10 秒钟，这个基本就是最后的采集周期耗时。

我们可以根据这些信息将交换机做一下分类，snmp 响应快的放在一起用一个 swcollector 采集，snmp 的超时时间可以设置的短一些，采集周期也可以设的小一点，比如30秒
响应慢的放在一起，用另外一个 swcollector 采集，snmp 的超时时间可以设置的长一点，采集周期也可以适当的放久一点，比如5分钟。

snmp 报文的响应需要消耗 cpu，因此交换机多少都对 snmp 报文的响应有速率限制，在 CPU 过高时，还可能会丢弃 snmp,icmp等优先级不高的请求。
不同品牌的对此的限定都有不同，有些型号可以通过配置修改。如果允许的话，也可以通过放开其对 snmp 的限速控制，来加快交换机对 snmp 报文的响应速度。

如此，在我们调节了 *“木桶” *中 *“木板”* 的长度后，swcollector 的单个实例可以很轻松的带起上百台乃至更多的交换机。


