package g

import (
	"time"
)

// changelog:
// 3.1.3: code refactor
// 3.1.4: bugfix ignore configuration
// 4.3.1: 修改交换机采集为多进程按端口方式采集，添加交换机端口丢包数据，添加本机网卡，iptables统计内外网流量数据，修改启动方式支持superior。
const (
	VERSION          = "4.3.1"
	COLLECT_INTERVAL = time.Second
)
