package tlstransport

import (
	"bytes"
	"errors"
	"io"
)

func (c *tlsConn) Read(p []byte) (int, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()
	if len(p) == 0 {
		return 0, nil
	}
	if !c.filterSettingsACK {
		return c.UConn.Read(p)
	}
	for {
		if len(c.frameHeader) > 0 {
			n := copy(p, c.frameHeader)
			c.frameHeader = c.frameHeader[n:]
			return n, nil
		}
		if c.frameRemaining > 0 {
			n, err := c.UConn.Read(p[:min(len(p), c.frameRemaining)])
			c.frameRemaining -= n
			return n, err
		}
		var header [9]byte
		if _, err := io.ReadFull(c.UConn, header[:]); err != nil {
			return 0, err
		}
		c.frameRemaining = int(header[0])<<16 | int(header[1])<<8 | int(header[2])
		if c.frameRemaining == 0 && header[3] == 4 && header[4] == 1 &&
			header[5]&0x7f == 0 && header[6] == 0 && header[7] == 0 && header[8] == 0 {
			c.settingsACKs++
			if c.settingsACKs == 2 {
				// 消费本层追加 SETTINGS 对应的 ACK，底层只跟踪自己的首个 SETTINGS。
				c.filterSettingsACK = false
				return c.UConn.Read(p)
			}
		}
		c.frameHeader = header[:]
	}
}

func (c *tlsConn) Write(p []byte) (int, error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if !c.limitHTTP2Window {
		return c.UConn.Write(p)
	}
	// req v3.54 的流接收缓冲固定为 4 MiB。保留 Profile 的首个 SETTINGS，
	// 随即在任何 HEADERS 之前降低有效窗口，避免慢消费触发 FLOW_CONTROL_ERROR。
	// 这是可观测的额外 SETTINGS，不能声称整个连接与浏览器逐字节相同。
	const preface = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
	c.initialWrite = append(c.initialWrite, p...)
	if len(c.initialWrite) < len(preface)+9 {
		return len(p), nil
	}
	data := c.initialWrite
	if !bytes.HasPrefix(data, []byte(preface)) || data[len(preface)+3] != 4 {
		return 0, errors.New("tlstransport: HTTP/2 初始化报文无效")
	}
	frameLen := int(data[24])<<16 | int(data[25])<<8 | int(data[26])
	end := len(preface) + 9 + frameLen
	if len(data) < end {
		return len(p), nil
	}
	// SETTINGS_INITIAL_WINDOW_SIZE = 4194304，帧载荷为 6 字节。
	adjustment := []byte{0, 0, 6, 4, 0, 0, 0, 0, 0, 0, 4, 0, 64, 0, 0}
	wire := make([]byte, 0, len(data)+len(adjustment))
	wire = append(wire, data[:end]...)
	wire = append(wire, adjustment...)
	wire = append(wire, data[end:]...)
	c.initialWrite = nil
	c.limitHTTP2Window = false
	n, err := c.UConn.Write(wire)
	if err != nil {
		return 0, err
	}
	if n != len(wire) {
		return 0, io.ErrShortWrite
	}
	return len(p), nil
}
