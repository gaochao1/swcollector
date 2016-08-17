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
// 3.2.1 add Discards,Error,UnknownProtos,QLen，fix some bugs
<<<<<<< HEAD
// 3.2.1.1 debugmetric support multi endpoints and metrics
const (
	VERSION          = "3.2.1.1"
=======
const (
	VERSION          = "3.2.1"
>>>>>>> origin/master
	COLLECT_INTERVAL = time.Second
)
