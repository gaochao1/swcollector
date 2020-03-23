package g

import (
	"bufio"
	"io"
	"log"
	"net"
	"net/rpc"
	"reflect"
	"time"

	"github.com/didi/nightingale/src/dataobj"

	"github.com/open-falcon/common/model"
	"github.com/ugorji/go/codec"
)

func N9ePush(items []*model.MetricValue) {
	var mh codec.MsgpackHandle
	mh.MapType = reflect.TypeOf(map[string]interface{}(nil))

	addr := config.Transfer.Addr
	retry := 0
	for {
		conn, err := net.DialTimeout("tcp", addr, time.Millisecond*3000)
		if err != nil {
			log.Println("dial transfer err:", err)
			continue
		}

		var bufconn = struct { // bufconn here is a buffered io.ReadWriteCloser
			io.Closer
			*bufio.Reader
			*bufio.Writer
		}{conn, bufio.NewReader(conn), bufio.NewWriter(conn)}

		rpcCodec := codec.MsgpackSpecRpc.ClientCodec(bufconn, &mh)
		client := rpc.NewClientWithCodec(rpcCodec)

		var reply dataobj.TransferResp
		err = client.Call("Transfer.Push", items, &reply)
		client.Close()
		if err != nil {
			log.Println(err)
			continue
		} else {
			if reply.Msg != "ok" {
				log.Println("some item push err", reply)
			}
			return
		}
		time.Sleep(time.Millisecond * 500)

		retry += 1
		if retry == 3 {
			retry = 0
			break
		}
	}
}
