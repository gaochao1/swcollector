# Changelog #
## 3.2.1.1 ##
1. debugmetric 现在支持配置多个 endpoint 和 metric 了
## 3.2.1 ##
#### 新功能 ####
1. 增加接口丢包数量的采集
	* IfInDiscards
	* IfOutDiscards

2. 增加接口错包数量的采集
	* IfInErrors
	* IfOutErros
	
3. 增加接口由于未知或不支持协议造成的丢包数量采集
	* IfInUnknownProtos
	
4. 增加接口输出队列中的包数量采集
	* IfOutQLen

5. 现在能够通过 debugmetric 配置选项，配置想要 debug 的 metric 了。配置后日志中会具体打印该条 metric 的日志

6. 现在能够通过 gosnmp 的配置选项,选择采用 gosnmp 还是 snmpwalk 进行数据采集了。两者效率相仿，snmpwalk稍微多占用点系统资源

### 改进 ###
1. 优化了 gosnmp 的端口采集，略微控制了一下并发速率，现在 gosnmp 采集端口超时的概率，应该有所降低了
2. 代码优化，删除了部分无关代码（比如 hbs 相关的部分……)
3. 部分日志的输出可读性更强了

#### bug修复 ####
1. 修复了一个广播报文采集不正确的 bug
2. 修复了一个老版本思科交换机，CPU 内存采集不正确的 bug
3. 修复了一些偶尔进程崩溃的 bug

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

4. 内置了更多交换机型号的 CPU， 内存的 OID 和计算方式。（锐捷，Juniper, 华为， 华三的一些型号等)

PS: 虽然 if 采集是并发的，不过采集项开的太多还是可能会影响 snmp 的采集效率，尤其是华为等 snmp 返回比较慢的交换机…………故谨慎选择，按需开启。

#### 改进 ####
1. 解决了 if 采集乱序的问题，现在即便使用 gosnmp 采集返回乱序也可以正确处理了。已测试过的华为型号现在均使用 gosnmp 采集。（v5.13，v5.70，v3.10）
2. 现在 log 中 打印 panic 信息的时候，应该会带上具体的 ip 地址了。
3. 现在默认采集 bit 单位的网卡流量了。
4. 去掉了默认配置文件里的 hostname 和 ip 选项，以免产生歧义，反正也没什么用…………
5. 修改默认 http 端口为 1989，避免和 agent 的端口冲突。

PS: func/swifstat.go 151行的注释代码，会在 debug 模式下打印具体的 ifstat 输出。如果交换机采集数据出现不准确的情况，可开启这段代码来进行排查。

#### bug修复 ####
1. 修复了在并发 ping 的情况下，即便 ip 地址不通，也有小概率 ping 通地址的 bug。（很神奇是不是……反正在我这里有出现这现象。。。）。方案是替换为 [go-fastping](https://github.com/tatsushid/go-fastping) 来做 ping 探测，通过 fastPingMode 配置选项开启。
2. 修复了思科 ASA-5585 9.1 和 9.2 两个版本 cpu, memory 的 oid 不一致带来的采集问题。（这坑爹玩意!)。现在应该可以根据他的版本号来选择不同的 oid 进行采集了。