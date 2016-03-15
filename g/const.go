package g

import (
	"time"
)

// changelog:
// 3.1.3: code refactor
// 3.1.4: bugfix ignore configuration
// 3.1.5: more sw support, DisplayByBit cfg
// 3.1.6
// 3.2.0: more sw support, fix ping bug, add ifOperStatus,ifBroadcastPkt,ifMulticastPkt
const (
	VERSION          = "3.2.0"
	COLLECT_INTERVAL = time.Second
)
