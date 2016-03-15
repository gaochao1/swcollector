

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

##源码安装
	依赖$GOPATH/src/github.com/gaochao1/sw
	cd $GOPATH/src/github.com/gaochao1/swcollector
	go get ./...
	./control build
	./control pack
	最后一步会pack出一个tar.gz的安装包，拿着这个包去部署服务即可。
	
	升级时，确保先更新sw
	cd $GOPATH/src/github.com/gaochao1/sw
	git pull

##部署说明

swcollector需要部署到有交换机SNMP访问权限的服务器上。

使用Go原生的ICMP协议进行Ping探测，swcollector需要root权限运行。

部分交换机使用Go原生SNMP协议会超时。暂时解决方法是SNMP接口流量查询前先判断设备型号，对部分此类设备，调用snmpwalk命令进行数据收集。(一些华为设备和思科的IOS XR)
因此最好在监控探针服务器上也装个snmpwalk命令


#配置说明

配置文件请参照cfg.example.json，修改该文件名为cfg.json，将该文件里的IP换成实际使用的IP。

switch配置项说明：

	"switch":{
	   "enabled": true,          
		"ipRange":[						#交换机IP地址段，对该网段有效IP，先发Ping包探测，对存活IP发SNMP请求
           "192.168.1.0/24",      
           "192.168.56.102/32",
           "172.16.114.233" 
 		],
 		"pingTimeout":300, 			   #Ping超时时间，单位毫秒
		"pingRetry":4,				   #Ping探测重试次数
		"community":"public",			#SNMP认证字符串
		"snmpTimeout":2000,				#SNMP超时时间，单位毫秒
		"snmpRetry":5,					#SNMP重试次数
		"ignoreIface": ["Nu","NU","Vlan","Vl","LoopBack"],    #忽略的接口，如Nu匹配ifName为*Nu*的接口
		"ignorePkt": true,            #不采集IfHCInUcastPkts和IfHCOutUcastPkts
		"ignoreBroadcastPkt": true,   #不采集IfHCInBroadcastPkts和IfHCOutBroadcastPkts
		"ignoreMulticastPkt": true,   #不采集IfHCInMulticastPkts和IfHCOutMulticastPkts
		"ignoreOperStatus": true,     #不采集IfOperStatus
		"displayByBit": true,		  #true时，上报的流量单位为bit，为false则单位为byte。
		"fastPingMode": false,	      #是否开启 fastPing 模式，开启 Ping 的效率更高，并能解决高并发时，会有小概率 ping 通宕机的交换机地址的情况。但 fastPing 可能被防火墙过滤。	
 		"limitConcur": 1000           #限制SNMP请求并发数
    }


