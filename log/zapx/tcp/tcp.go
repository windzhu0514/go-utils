package tcp

import (
	"net"

	gas "github.com/netbrain/goautosocket"
	"go.uber.org/zap/zapcore"
)

// https://github.com/bshuster-repo/logrus-logstash-hook
// [Reconnect to logstash on connection timeout](https://github.com/bshuster-repo/logrus-logstash-hook/issues/48)
// [Logstash hook for logrus with async mode and reconnects](https://github.com/iost-official/logrustash)
// [The GAS library provides auto-reconnecting TCP sockets in a tiny, fully tested, thread-safe API](https://github.com/netbrain/goautosocket)

type tcp struct {
	conn net.Conn
}

func New(addr string) (zapcore.WriteSyncer, error) {
	ws := &tcp{}
	var err error
	ws.conn, err = gas.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	return ws, nil
}

func (ws *tcp) Write(p []byte) (n int, err error) {
	return ws.conn.Write(p)
}

func (ws *tcp) Sync() error {
	return nil
}
