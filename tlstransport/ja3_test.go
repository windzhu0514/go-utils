package tlstransport

import (
	"crypto/md5"
	"fmt"
	"testing"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/crypto/cryptobyte"
)

func TestUnitJA3_WireAndMalformed(t *testing.T) {
	var b cryptobyte.Builder
	b.AddUint8(1)
	b.AddUint24LengthPrefixed(func(b *cryptobyte.Builder) {
		b.AddUint16(771)
		b.AddBytes(make([]byte, 32))
		b.AddUint8(0)
		b.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) {
			b.AddUint16(0x0a0a)
			b.AddUint16(4865)
			b.AddUint16(4866)
		})
		b.AddUint8(1)
		b.AddUint8(0)
		b.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) {
			b.AddUint16(0x1a1a)
			b.AddUint16(0)
			b.AddUint16(10)
			b.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) {
				b.AddUint16(6)
				b.AddUint16(0x2a2a)
				b.AddUint16(29)
				b.AddUint16(23)
			})
			b.AddUint16(11)
			b.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) { b.AddUint8(1); b.AddUint8(0) })
			b.AddUint16(0)
			b.AddUint16(0)
		})
	})
	raw, err := b.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	want := "771,4865-4866,10-11-0,29-23,0"
	value, hash, err := JA3FromClientHello(raw)
	if err != nil || value != want || hash != fmt.Sprintf("%x", md5.Sum([]byte(want))) {
		t.Fatalf("JA3 解析错误: %s %s %v", value, hash, err)
	}
	for i := range len(raw) {
		if _, _, err := JA3FromClientHello(raw[:i]); err == nil {
			t.Fatalf("截断消息未报错: %d", i)
		}
	}
}

func TestUnitJA3_ServerNameAndStableProfile(t *testing.T) {
	tr := newTestTransport(t, WithClientHelloID(utls.HelloChrome_100))
	first, _, err := tr.JA3("example.com")
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := tr.JA3("example.com")
	if err != nil || first != second {
		t.Fatalf("固定顺序指纹不稳定: %s %s %v", first, second, err)
	}
	ipJA3, _, err := tr.JA3("127.0.0.1")
	if err != nil || first == ipJA3 {
		t.Fatalf("IP 地址不应发送 SNI 扩展: %s %v", ipJA3, err)
	}
}
