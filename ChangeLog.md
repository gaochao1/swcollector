# Changelog #
## 3.2.0 ##
#### 新功能 ####
1. 增加接口广播包数量的采集
	* IfHCInBroadcastPkts
	* IfHCOutBroadcastPkts

2. 增加接口组播包数量的采集
	* IfHCInMulticastPkts
	* IfHCOutMulticastPkts

3. 增加接口状态的采集
	* IfOperStatus(1 up, 2 down, 3 testing, 4 unknown, 5 dormant, 6 notPresent, 7 lowerLayerDown)

虽然采集是并发的，不过采集项开的太多还是可能会影响 snmp 的采集效率，尤其是华为等 snmp 返回比较慢的交换机…………故谨慎选择，按需开启。

内置了更多交换机型号的CPU，内存的OID和计算方式。（锐捷，Juniper,华为，华三的一些型号等)

#### 改进 ####
1. 解决了 if 采集乱序的问题，现在即便使用 gosnmp 采集返回乱序也可以正确处理了。已测试过的华为型号现在均使用 gosnmp 采集。（v5.13，v5.70，v3.10）
2. 现在 log 中 打印 panic 信息的时候，应该会带上具体的 ip 地址了。
3. 现在默认采集 bit 单位的网卡流量了。
4. 去掉了默认配置文件里的 hostname 和 ip 选项，以免产生歧义，反正也没什么用…………

#### bug修复 ####
1. 修复了在并发 ping 的情况下，即便 ip 地址不同，也有小概率 ping 通地址的 bug。（很神奇是不是……反正在我这里有出现这现象。。。）。方案是替换为 [go-fastping](https://github.com/tatsushid/go-fastping) 来做 ping 探测。
2. 修复了思科 ASA-5585 9.1 和 9.2 两个版本 cpu, memory 的 oid 不一致带来的采集问题。（这坑爹玩意!)。现在应该可以根据他的版本号来选择不同的 oid 进行采集了。