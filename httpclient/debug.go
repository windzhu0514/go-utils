package httpclient

import "sync"

type Debug struct {
	mu               sync.Mutex
	TlsConnCount     map[string]int64 // 此host建立过多少个tls链接
	TlsHandShakeTime map[string]int64 // 与host建立tls链接的tls握手时延, 毫秒
}

func (d *Debug) IncrTlsConnCount(host string) {
	d.mu.Lock()
	count := d.TlsConnCount[host]
	count++
	d.TlsConnCount[host] = count
	d.mu.Unlock()
}

func (d *Debug) GetTlsConnCount(host string) int64 {
	d.mu.Lock()
	count := d.TlsConnCount[host]
	d.mu.Unlock()
	return count
}

func (d *Debug) SetTlsHandShakeTime(host string, t int64) {
	d.mu.Lock()
	d.TlsHandShakeTime[host] = t
	d.mu.Unlock()
	return
}

func (d *Debug) GetTlsHandShakeTime(host string) int64 {
	d.mu.Lock()
	t := d.TlsHandShakeTime[host]
	d.mu.Unlock()
	return t
}
