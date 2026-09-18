package tlstransport

import (
	"crypto/md5" // JA3 定义使用 MD5，仅作为指纹标识，不用于安全校验。
	"encoding/binary"
	"encoding/hex"
	"errors"
	"net"
	"strconv"
	"strings"

	"golang.org/x/crypto/cryptobyte"
)

// JA3 离线生成一次 ClientHello 预览，返回 JA3 原文及 MD5。
// 默认使用 example.com 作为 SNI，可传入实际服务器名；最多一个参数。
// 随机扩展顺序、会话恢复和不同 SNI 会影响实际握手，预览不代表线上的固定值。
func (t *Transport) JA3(serverNames ...string) (string, string, error) {
	if len(serverNames) > 1 {
		return "", "", errors.New("tlstransport: JA3 最多接受一个服务器名")
	}
	serverName := "example.com"
	if len(serverNames) == 1 {
		serverName = serverNames[0]
	}
	local, remote := net.Pipe()
	defer local.Close()
	defer remote.Close()
	conn, err := t.newConn(local, serverName)
	if err != nil {
		return "", "", err
	}
	if err := conn.BuildHandshakeState(); err != nil {
		return "", "", err
	}
	return JA3FromClientHello(conn.HandshakeState.Hello.Raw)
}

// JA3FromClientHello 从完整 TLS ClientHello 握手消息计算 JA3（过滤 GREASE）。
// raw 必须包含 4 字节握手头，不包含 TLS record 头；跨 record 的数据须先重组。
func JA3FromClientHello(raw []byte) (string, string, error) {
	invalid := errors.New("tlstransport: ClientHello 编码无效")
	input := cryptobyte.String(raw)
	var messageType uint8
	var hello cryptobyte.String
	if !input.ReadUint8(&messageType) || messageType != 1 || !input.ReadUint24LengthPrefixed(&hello) || !input.Empty() {
		return "", "", invalid
	}
	var version uint16
	var session, ciphers, compression cryptobyte.String
	if !hello.ReadUint16(&version) || !hello.Skip(32) || !hello.ReadUint8LengthPrefixed(&session) ||
		!hello.ReadUint16LengthPrefixed(&ciphers) || len(ciphers) == 0 || len(ciphers)%2 != 0 ||
		!hello.ReadUint8LengthPrefixed(&compression) || len(compression) == 0 {
		return "", "", invalid
	}
	var extensions, groups, points []string
	if !hello.Empty() {
		var all cryptobyte.String
		if !hello.ReadUint16LengthPrefixed(&all) || !hello.Empty() {
			return "", "", invalid
		}
		for !all.Empty() {
			var id uint16
			var data cryptobyte.String
			if !all.ReadUint16(&id) || !all.ReadUint16LengthPrefixed(&data) {
				return "", "", invalid
			}
			if !isGREASE(id) {
				extensions = append(extensions, strconv.Itoa(int(id)))
			}
			switch id {
			case 10:
				var curves cryptobyte.String
				if !data.ReadUint16LengthPrefixed(&curves) || !data.Empty() || len(curves)%2 != 0 {
					return "", "", invalid
				}
				groups = decimalIDs(curves)
			case 11:
				var formats cryptobyte.String
				if !data.ReadUint8LengthPrefixed(&formats) || !data.Empty() {
					return "", "", invalid
				}
				for _, p := range formats {
					points = append(points, strconv.Itoa(int(p)))
				}
			}
		}
	}
	value := strings.Join([]string{strconv.Itoa(int(version)), strings.Join(decimalIDs(ciphers), "-"),
		strings.Join(extensions, "-"), strings.Join(groups, "-"), strings.Join(points, "-")}, ",")
	sum := md5.Sum([]byte(value))
	return value, hex.EncodeToString(sum[:]), nil
}

func decimalIDs(data []byte) []string {
	var result []string
	for len(data) >= 2 {
		id := binary.BigEndian.Uint16(data)
		if !isGREASE(id) {
			result = append(result, strconv.Itoa(int(id)))
		}
		data = data[2:]
	}
	return result
}

func isGREASE(id uint16) bool {
	return id&0x0f0f == 0x0a0a && byte(id) == byte(id>>8)
}
