package httpclient

import (
	"fmt"
	"time"

	tls "github.com/refraction-networking/utls"
)

func GetCustomTlsSpec() *tls.ClientHelloSpec {
	now := time.Now().Unix()
	length := len(TlsCustomSpec)
	num := now % int64(length)
	// num = 18
	fmt.Println("change spec: ", num)
	return TlsCustomSpec[num]
}

// 自定义tls指纹
var TlsCustomSpec = []*tls.ClientHelloSpec{
	CustomClientHelloShuffleCipherSuitesSpec01,
	CustomClientHelloShuffleCipherSuitesSpec02,
}

// 打乱CipherSuites顺序, 删除
var CustomClientHelloShuffleCipherSuitesSpec01 = &tls.ClientHelloSpec{
	TLSVersMax: tls.VersionTLS12,
	TLSVersMin: tls.VersionTLS10,
	CipherSuites: []uint16{
		tls.GREASE_PLACEHOLDER,
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
	},
	CompressionMethods: []byte{
		0x00, // compressionNone
	},
	Extensions: []tls.TLSExtension{
		&tls.UtlsGREASEExtension{},
		&tls.SNIExtension{},
		//&tls.UtlsExtendedMasterSecretExtension{},
		&tls.RenegotiationInfoExtension{Renegotiation: tls.RenegotiateOnceAsClient},
		&tls.SupportedCurvesExtension{[]tls.CurveID{
			tls.CurveID(tls.GREASE_PLACEHOLDER),
			tls.X25519,
			tls.CurveP256,
		}},
		&tls.SupportedPointsExtension{SupportedPoints: []byte{
			0x00, // pointFormatUncompressed
		}},
		&tls.SessionTicketExtension{},
		&tls.ALPNExtension{AlpnProtocols: []string{"h2", "http/1.1"}},
		&tls.StatusRequestExtension{},
		&tls.SignatureAlgorithmsExtension{SupportedSignatureAlgorithms: []tls.SignatureScheme{
			tls.ECDSAWithP256AndSHA256,
			tls.PSSWithSHA256,
			tls.PKCS1WithSHA256,
			tls.ECDSAWithP384AndSHA384,
			tls.PSSWithSHA384,
			tls.PKCS1WithSHA384,
			tls.PSSWithSHA512,
			tls.PKCS1WithSHA512,
		}},
		&tls.SCTExtension{},
		&tls.KeyShareExtension{[]tls.KeyShare{
			// {Group: tls.CurveID(tls.GREASE_PLACEHOLDER), Data: []byte{0}},
			{Group: tls.X25519},
		}},
		&tls.PSKKeyExchangeModesExtension{[]uint8{
			tls.PskModeDHE,
		}},
		&tls.SupportedVersionsExtension{[]uint16{
			tls.GREASE_PLACEHOLDER,
			tls.VersionTLS12,
			tls.VersionTLS11,
			tls.VersionTLS10,
		}},
		&tls.UtlsCompressCertExtension{[]tls.CertCompressionAlgo{
			tls.CertCompressionBrotli,
		}},
		&tls.UtlsGREASEExtension{},
		&tls.UtlsPaddingExtension{GetPaddingLen: tls.BoringPaddingStyle},
	},
	GetSessionID: nil,
}

var CustomClientHelloShuffleCipherSuitesSpec02 = &tls.ClientHelloSpec{
	TLSVersMax: tls.VersionTLS12,
	TLSVersMin: tls.VersionTLS10,
	CipherSuites: []uint16{
		tls.GREASE_PLACEHOLDER,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
	},
	CompressionMethods: []byte{
		0x00, // compressionNone
	},
	Extensions: []tls.TLSExtension{
		&tls.UtlsGREASEExtension{},
		&tls.SNIExtension{},
		//&tls.UtlsExtendedMasterSecretExtension{},
		&tls.RenegotiationInfoExtension{Renegotiation: tls.RenegotiateOnceAsClient},
		&tls.SupportedCurvesExtension{[]tls.CurveID{
			tls.CurveID(tls.GREASE_PLACEHOLDER),
			tls.X25519,
			tls.CurveP256,
		}},
		&tls.SupportedPointsExtension{SupportedPoints: []byte{
			0x00, // pointFormatUncompressed
		}},
		&tls.SessionTicketExtension{},
		&tls.ALPNExtension{AlpnProtocols: []string{"http/1.1", "h2"}},
		&tls.StatusRequestExtension{},
		&tls.SignatureAlgorithmsExtension{SupportedSignatureAlgorithms: []tls.SignatureScheme{
			tls.ECDSAWithP256AndSHA256,
			tls.PSSWithSHA256,
			tls.PKCS1WithSHA256,
			tls.ECDSAWithP384AndSHA384,
			tls.PSSWithSHA384,
			tls.PKCS1WithSHA384,
			tls.PSSWithSHA512,
			tls.PKCS1WithSHA512,
		}},
		&tls.SCTExtension{},
		&tls.KeyShareExtension{[]tls.KeyShare{
			{Group: tls.CurveID(tls.GREASE_PLACEHOLDER), Data: []byte{0}},
			{Group: tls.X25519},
		}},
		&tls.PSKKeyExchangeModesExtension{[]uint8{
			tls.PskModeDHE,
		}},
		&tls.SupportedVersionsExtension{[]uint16{
			tls.GREASE_PLACEHOLDER,
			tls.VersionTLS12,
			tls.VersionTLS11,
			tls.VersionTLS10,
		}},
		&tls.UtlsCompressCertExtension{[]tls.CertCompressionAlgo{
			tls.CertCompressionBrotli,
		}},
		&tls.UtlsGREASEExtension{},
		&tls.UtlsPaddingExtension{GetPaddingLen: tls.BoringPaddingStyle},
	},
	GetSessionID: nil,
}
