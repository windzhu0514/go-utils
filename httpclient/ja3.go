package httpclient

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"

	tls "github.com/refraction-networking/utls"
)

type Ja3Slice []uint16

func (ja3Slice *Ja3Slice) ToString() string {
	slice := *ja3Slice
	arrStr := make([]string, len(slice))
	for i, num := range slice {
		arrStr[i] = strconv.Itoa(int(num))
	}

	return strings.Join(arrStr, "-")
}

// 获取ja3指纹
func (transpoort *Transport) JA3() (string, string) {
	var (
		ja3              string
		ja3Md5           string
		ja3Cipher        *Ja3Slice
		ja3Extentions    *Ja3Slice
		ja3SupportGroups *Ja3Slice
		ja3Points        *Ja3Slice
		clientHelloSpec  *tls.ClientHelloSpec
	)
	if transpoort.ClientHelloID == tls.HelloCustom {
		if transpoort.ClientHelloSpec == nil {
			fmt.Println("Custome client hello spec is nil")
			return "", ""
		}
		clientHelloSpec = transpoort.ClientHelloSpec
	} else {
		clientHelloStruct, err := tls.UTLSIdToSpec(transpoort.ClientHelloID)
		if err != nil {
			fmt.Println("UTLSIdToSpec error: ", err)
			return "", ""
		}
		clientHelloSpec = &clientHelloStruct
	}
	ja3Cipher = transpoort.getCipherSuite(clientHelloSpec)
	ja3Extentions = transpoort.getExtensions(clientHelloSpec)
	ja3SupportGroups = transpoort.getSupportGroup(clientHelloSpec)
	ja3Points = transpoort.getSupportedPoints(clientHelloSpec)
	tlsVersion := 0x0303 // 一般都是使用tls1.2
	ja3 = fmt.Sprintf("%d,%s,%s,%s,%s", tlsVersion, ja3Cipher.ToString(),
		ja3Extentions.ToString(), ja3SupportGroups.ToString(), ja3Points.ToString())
	md := md5.New()
	md.Write([]byte(ja3))
	ja3Md5 = hex.EncodeToString(md.Sum(nil))
	return ja3, ja3Md5
}

func (transpoort *Transport) getCipherSuite(clientHello *tls.ClientHelloSpec) *Ja3Slice {
	res := make(Ja3Slice, 0, len(clientHello.CipherSuites))
	for _, i := range clientHello.CipherSuites {
		if isGREASEUint16(i) {
			continue
		}
		res = append(res, i)
	}
	return &res
}

func (transpoort *Transport) getSupportedPoints(clientHello *tls.ClientHelloSpec) *Ja3Slice {
	res := make(Ja3Slice, 0, 10)
	for _, extension := range clientHello.Extensions {
		if ext, ok := extension.(*tls.SupportedPointsExtension); ok {
			for _, point := range ext.SupportedPoints {
				res = append(res, uint16(point))
			}

		}
	}
	return &res
}

func (transpoort *Transport) getSupportGroup(clientHello *tls.ClientHelloSpec) *Ja3Slice {
	res := make(Ja3Slice, 0, 10)
	for _, extension := range clientHello.Extensions {
		if ext, ok := extension.(*tls.SupportedCurvesExtension); ok {
			for _, curveID := range ext.Curves {
				if isGREASEUint16(uint16(curveID)) {
					continue
				}
				res = append(res, uint16(curveID))
			}

		}
	}
	return &res
}

// 有些扩展需要特殊区分处理，如果ja3和wireshark不一样，需要看一下具体哪个扩展不一样
func (transpoort *Transport) getExtensions(clientHello *tls.ClientHelloSpec) *Ja3Slice {
	res := make(Ja3Slice, 0, 20)
	for _, extention := range clientHello.Extensions {
		var num uint16
		switch extention.(type) {
		case *tls.UtlsPaddingExtension:
			num = uint16(21)
		case *tls.UtlsGREASEExtension:
			continue
		case *tls.SNIExtension:
			num = uint16(0)
		default:
			data := make([]byte, extention.Len())
			_, err := extention.Read(data)
			if err != nil && err != io.EOF {
				fmt.Println("tlsConnn getExtensions get wrong extention")
				continue
			}
			if len(data) == 0 {
				continue
			}
			num = uint16(data[0])<<8 | uint16(data[1])
		}
		res = append(res, num)
	}
	return &res
}

// tls用于扩展的字段，不需要参与ja3计算
func isGREASEUint16(v uint16) bool {
	return ((v >> 8) == v&0xff) && v&0xf == 0xa
}
