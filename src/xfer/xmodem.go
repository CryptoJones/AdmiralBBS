// Package xfer implements XMODEM (CRC-16) file transfer over a caller's
// session — the authentic BBS way to move binary files. Send streams a file to
// the caller's terminal program; Receive accepts an upload.
package xfer

import (
	"errors"
	"io"
)

const (
	soh = 0x01 // 128-byte block start
	eot = 0x04 // end of transmission
	ack = 0x06
	nak = 0x15
	can = 0x18
	sub = 0x1A // padding byte
	crc = 'C'  // receiver requests CRC mode
)

var (
	// ErrCanceled means the peer sent CAN.
	ErrCanceled = errors.New("xfer: transfer cancelled by peer")
	// ErrNoStart means the receiver never requested the transfer.
	ErrNoStart = errors.New("xfer: receiver did not start")
	// ErrNoAck means the peer stopped acknowledging.
	ErrNoAck = errors.New("xfer: no acknowledgement from peer")
)

const maxRetries = 10

// crc16 is the CRC-16/XMODEM checksum (poly 0x1021, init 0x0000).
func crc16(data []byte) uint16 {
	var c uint16
	for _, b := range data {
		c ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if c&0x8000 != 0 {
				c = c<<1 ^ 0x1021
			} else {
				c <<= 1
			}
		}
	}
	return c
}

func readByte(r io.Reader) (byte, error) {
	var b [1]byte
	_, err := io.ReadFull(r, b[:])
	return b[0], err
}

// Send transmits data to the receiver using XMODEM-CRC, padding the final block.
func Send(rw io.ReadWriter, data []byte) error {
	// Wait for the receiver to request CRC mode ('C') or checksum mode (NAK).
	useCRC := true
	for tries := 0; ; tries++ {
		b, err := readByte(rw)
		if err != nil {
			return err
		}
		if b == crc {
			useCRC = true
			break
		}
		if b == nak {
			useCRC = false
			break
		}
		if b == can {
			return ErrCanceled
		}
		if tries >= maxRetries {
			return ErrNoStart
		}
	}

	block := byte(1)
	for off := 0; off < len(data) || off == 0; off += 128 {
		var chunk [128]byte
		n := copy(chunk[:], data[min(off, len(data)):])
		for i := n; i < 128; i++ {
			chunk[i] = sub
		}
		if err := sendBlock(rw, block, chunk[:], useCRC); err != nil {
			return err
		}
		block++
		if off+128 >= len(data) {
			break
		}
	}

	for tries := 0; tries < maxRetries; tries++ {
		if _, err := rw.Write([]byte{eot}); err != nil {
			return err
		}
		b, err := readByte(rw)
		if err != nil {
			return err
		}
		if b == ack {
			return nil
		}
	}
	return ErrNoAck
}

func sendBlock(rw io.ReadWriter, block byte, chunk []byte, useCRC bool) error {
	frame := make([]byte, 0, 3+128+2)
	frame = append(frame, soh, block, 255-block)
	frame = append(frame, chunk...)
	if useCRC {
		c := crc16(chunk)
		frame = append(frame, byte(c>>8), byte(c))
	} else {
		var sum byte
		for _, b := range chunk {
			sum += b
		}
		frame = append(frame, sum)
	}
	for tries := 0; tries < maxRetries; tries++ {
		if _, err := rw.Write(frame); err != nil {
			return err
		}
		b, err := readByte(rw)
		if err != nil {
			return err
		}
		switch b {
		case ack:
			return nil
		case can:
			return ErrCanceled
		case nak:
			continue // resend
		}
	}
	return ErrNoAck
}

// Receive accepts an XMODEM-CRC upload and returns the data (trailing SUB
// padding trimmed). It drives CRC mode by sending 'C'.
func Receive(rw io.ReadWriter) ([]byte, error) {
	if _, err := rw.Write([]byte{crc}); err != nil {
		return nil, err
	}
	var out []byte
	expected := byte(1)
	for {
		b, err := readByte(rw)
		if err != nil {
			return nil, err
		}
		switch b {
		case eot:
			rw.Write([]byte{ack})
			return trimPad(out), nil
		case can:
			return nil, ErrCanceled
		case soh:
			hdr := make([]byte, 2)
			if _, err := io.ReadFull(rw, hdr); err != nil {
				return nil, err
			}
			data := make([]byte, 128)
			if _, err := io.ReadFull(rw, data); err != nil {
				return nil, err
			}
			ck := make([]byte, 2)
			if _, err := io.ReadFull(rw, ck); err != nil {
				return nil, err
			}
			goodHdr := hdr[0] == 255-hdr[1]
			goodCRC := crc16(data) == uint16(ck[0])<<8|uint16(ck[1])
			switch {
			case goodHdr && goodCRC && hdr[0] == expected:
				out = append(out, data...)
				expected++
				rw.Write([]byte{ack})
			case goodHdr && hdr[0] == expected-1:
				rw.Write([]byte{ack}) // duplicate block; re-ACK, don't append
			default:
				rw.Write([]byte{nak})
			}
		}
	}
}

func trimPad(b []byte) []byte {
	i := len(b)
	for i > 0 && b[i-1] == sub {
		i--
	}
	return b[:i]
}
