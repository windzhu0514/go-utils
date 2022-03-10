package hash

import (
	"errors"
)

// https://crccalc.com/
// http://www.ip33.com/crc.html
// https://github.com/sigurn/crc16/blob/master/crc16.go
// https://github.com/howeyc/crc16/blob/master/crc16.go
// https://github.com/snksoft/crc/blob/master/crc.go
// https://github.com/stellar/go/blob/master/crc16/main.go

// The size of a CRC-16 checksum in bytes.
const Size = 2

// Predefined polynomials.
const (
	IBM         = 0xA001 // IBM 16-bit
	ARC         = 0x8005 // ARC
	AUG_CCITT   = 0x1021 // AUG-CCITT
	BUYPASS     = 0x8005 // BUYPASS
	CCITT_FALSE = 0x1021 // CCITT-FALSE
	CDMA2000    = 0xC867 // CDMA2000
	DDS_110     = 0x8005 // DDS-110
	DECT_R      = 0x0589 // DECT-R
	DECT_X      = 0x0589 // DECT-X
	DNP         = 0x3D65 // DNP
	EN_13757    = 0x3D65 // EN-13757
	GENIBUS     = 0x1021 // GENIBUS
	MAXIM       = 0x8005 // MAXIM
	MCRF4XX     = 0x1021 // MCRF4XX
	RIELLO      = 0x1021 // RIELLO
	T10_DIF     = 0x8BB7 // T10-DIF
	TELEDISK    = 0xA097 // TELEDISK
	TMS37157    = 0x1021 // TMS37157
	USB         = 0x8005 // USB
	CRC_A       = 0x1021 // CRC-A
	KERMIT      = 0x1021 // KERMIT
	MODBUS      = 0x8005 // MODBUS
	X_25        = 0x1021 // X-25
	XMODEM      = 0x1021 // XMODEM
)

// Table is a 256-word table representing the polynomial for efficient processing.
type Table struct {
	entries  [256]uint16
	reversed bool
	noXOR    bool
}

var IBMTable = makeTable(IBM)

func MakeTable(poly uint16) *Table {
	return makeTable(poly)
}

func makeTable(poly uint16) *Table {
	t := &Table{
		reversed: false,
	}
	for i := 0; i < 256; i++ {
		crc := uint16(i)
		for j := 0; j < 8; j++ {
			if crc&1 == 1 {
				crc = (crc >> 1) ^ poly
			} else {
				crc >>= 1
			}
		}
		t.entries[i] = crc
	}
	return t
}

// digest represents the partial evaluation of a checksum.
type digest struct {
	crc uint16
	tab *Table
}

// New creates a new hash.Hash16 computing the CRC-16 checksum using the
// polynomial represented by the Table. Its Sum method will lay the
// value out in big-endian byte order. The returned Hash16 also
// implements encoding.BinaryMarshaler and encoding.BinaryUnmarshaler to
// marshal and unmarshal the internal state of the hash.
func New(tab *Table) Hash16 {
	return &digest{0, tab}
}

func NewIBM() Hash16 { return New(IBMTable) }

func (d *digest) Size() int { return Size }

func (d *digest) BlockSize() int { return 1 }

func (d *digest) Reset() { d.crc = 0 }

const (
	magic         = "crc16\x01"
	marshaledSize = len(magic) + 2 + 2 + 1
)

func (d *digest) MarshalBinary() ([]byte, error) {
	b := make([]byte, 0, marshaledSize)
	b = append(b, magic...)
	b = appendUint16(b, tableSum(d.tab))
	b = appendUint16(b, d.crc)
	return b, nil
}

func (d *digest) UnmarshalBinary(b []byte) error {
	if len(b) < len(magic) || string(b[:len(magic)]) != magic {
		return errors.New("hash/crc32: invalid hash state identifier")
	}
	if len(b) != marshaledSize {
		return errors.New("hash/crc32: invalid hash state size")
	}
	if tableSum(d.tab) != readUint16(b[4:]) {
		return errors.New("hash/crc32: tables do not match")
	}
	d.crc = readUint16(b[8:])
	return nil
}

// Update returns the result of adding the bytes in p to the crc.
func Update(crc uint16, tab *Table, p []byte) uint16 {
	return update(crc, tab, p)
}

func (d *digest) Write(p []byte) (n int, err error) {
	d.crc = update(d.crc, d.tab, p)
	return len(p), nil
}

func (d *digest) Sum16() uint16 { return d.crc }

func (d *digest) Sum(in []byte) []byte {
	s := d.Sum16()
	return append(in, byte(s>>8), byte(s))
}

// Checksum returns the CRC-32 checksum of data
// using the polynomial represented by the Table.
func Checksum(data []byte, tab *Table) uint16 { return Update(0, tab, data) }

// ChecksumIBM returns the CRC-16 checksum of data
// using the IBM polynomial.
func ChecksumIBM(data []byte) uint16 { return Update(0, IBMTable, data) }

// simpleUpdate uses the simple algorithm to update the CRC, given a table that
// was previously computed using simpleMakeTable.
func update(crc uint16, tab *Table, p []byte) uint16 {
	crc = ^crc
	for _, v := range p {
		crc = tab.entries[byte(crc)^v] ^ (crc >> 8)
	}
	return ^crc
}

func appendUint16(b []byte, x uint16) []byte {
	a := [2]byte{
		byte(x >> 8),
		byte(x),
	}
	return append(b, a[:]...)
}

func readUint16(b []byte) uint16 {
	_ = b[1]
	return uint16(b[1]) | uint16(b[0])<<8
}

// tableSum returns the IBM checksum of table t.
func tableSum(t *Table) uint16 {
	var a [1024]byte
	b := a[:0]
	if t != nil {
		for _, x := range t.entries {
			b = appendUint16(b, x)
		}
	}
	return ChecksumIBM(b)
}
