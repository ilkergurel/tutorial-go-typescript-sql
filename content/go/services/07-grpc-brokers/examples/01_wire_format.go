package main

import (
	"encoding/binary"
	"fmt"
)

// Two of protobuf's wire types: they tell any reader how to skip a field.
const (
	wireVarint = 0 // int32, int64, uint64, bool, enum
	wireLen    = 2 // string, bytes, nested messages
)

func appendVarint(b []byte, field int, v uint64) []byte {
	b = binary.AppendUvarint(b, uint64(field)<<3|wireVarint) // the tag
	return binary.AppendUvarint(b, v)
}

func appendString(b []byte, field int, s string) []byte {
	b = binary.AppendUvarint(b, uint64(field)<<3|wireLen)
	b = binary.AppendUvarint(b, uint64(len(s))) // a byte count, not characters
	return append(b, s...)
}

// decodeV1 knows fields 1-3 of Order and skips anything else. It trusts its
// input; generated code also rejects truncated or malformed bytes.
func decodeV1(b []byte) {
	for len(b) > 0 {
		tag, n := binary.Uvarint(b)
		b = b[n:]
		field, wire := tag>>3, tag&7
		var num uint64
		var str []byte
		switch wire {
		case wireVarint:
			num, n = binary.Uvarint(b)
			b = b[n:]
		case wireLen:
			size, n := binary.Uvarint(b)
			str, b = b[n:n+int(size)], b[n+int(size):]
		default:
			fmt.Println("  wire type", wire, "is not handled in this sketch")
			return
		}
		switch field {
		case 1:
			fmt.Println("  id:", num)
		case 2:
			fmt.Printf("  customer: %q\n", str)
		case 3:
			fmt.Println("  amount_kurus:", num)
		default:
			fmt.Printf("  unknown field %d (wire type %d): skipped\n", field, wire)
		}
	}
}

func main() {
	// message Order { int64 id = 1; string customer = 2; int64 amount_kurus = 3; }
	var order []byte
	order = appendVarint(order, 1, 150)
	order = appendString(order, 2, "Ayşe")
	order = appendVarint(order, 3, 125000)
	fmt.Printf("protobuf %2d bytes: % x\n", len(order), order)
	js := `{"id":150,"customer":"Ayşe","amount_kurus":125000}`
	fmt.Printf("JSON     %2d bytes: %s\n", len(js), js)

	// Version 2 of the schema adds: string note = 4;
	v2 := appendString(order, 4, "kapıda ödeme")
	fmt.Println("a v1 reader decoding a v2 message:")
	decodeV1(v2)
}
