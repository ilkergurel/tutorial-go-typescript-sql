package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
)

// Reading is one packet from a (made-up) greenhouse sensor: 12 bytes, big-endian.
type Reading struct {
	Magic    [2]byte // "SL"
	Version  uint8
	Flags    uint8
	DeviceID uint32
	TempDeci int16  // tenths of a degree: -125 is -12.5 °C
	Humidity uint16 // tenths of a percent
}

func main() {
	pkt := []byte{'S', 'L', 1, 0, 0x00, 0x01, 0xE2, 0x40, 0xFF, 0x83, 0x02, 0x6C}

	// Field by field: explicit offsets, no reflection.
	fmt.Println("device id, big-endian:   ", binary.BigEndian.Uint32(pkt[4:8]))
	fmt.Println("device id, little-endian:", binary.LittleEndian.Uint32(pkt[4:8]))

	// The whole struct: fields are read in order, packed, with no padding.
	var r Reading
	if err := binary.Read(bytes.NewReader(pkt), binary.BigEndian, &r); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", r)
	fmt.Printf("%.1f °C, %.1f %% humidity, %d bytes on the wire\n",
		float64(r.TempDeci)/10, float64(r.Humidity)/10, binary.Size(r))

	// Encoding: Append (Go 1.23) builds the bytes without an io.Writer.
	r.TempDeci = 215
	out, err := binary.Append(nil, binary.BigEndian, r)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("% x\n", out)

	// A short packet is an error, not a half-filled struct.
	err = binary.Read(bytes.NewReader(pkt[:7]), binary.BigEndian, &r)
	fmt.Println("7 of 12 bytes:", err, errors.Is(err, io.ErrUnexpectedEOF))

	// Varints: small numbers take fewer bytes (protobuf uses the same scheme).
	for _, n := range []uint64{5, 300, 1 << 32} {
		fmt.Printf("uvarint %d -> % x\n", n, binary.AppendUvarint(nil, n))
	}
}
