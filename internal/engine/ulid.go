package engine

import (
	"crypto/rand"
	"time"
)

// Crockford base32 encoding alphabet (ULID spec).
const crockfordAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// NewULID generates a ULID string (26 characters).
// 48-bit timestamp (ms) + 80-bit random, encoded in Crockford base32.
// Uses crypto/rand for randomness, zero external dependencies.
func NewULID() string {
	ts := uint64(time.Now().UnixMilli())
	var b [16]byte

	// 48-bit timestamp (6 bytes)
	b[0] = byte(ts >> 40)
	b[1] = byte(ts >> 32)
	b[2] = byte(ts >> 24)
	b[3] = byte(ts >> 16)
	b[4] = byte(ts >> 8)
	b[5] = byte(ts)

	// 80-bit random (10 bytes)
	if _, err := rand.Read(b[6:]); err != nil {
		// Fallback: use time-based data only (extremely unlikely path)
		for i := 6; i < 16; i++ {
			b[i] = byte(ts >> ((i - 6) * 8))
		}
	}

	return encodeCrockford(b)
}

// encodeCrockford encodes 16 bytes (128 bits) into 26 Crockford base32 characters.
func encodeCrockford(src [16]byte) string {
	var dst [26]byte

	// Encode 10 bytes of timestamp (80 bits → 16 chars)
	dst[0] = crockfordAlphabet[(src[0]&0b11111000)>>3]
	dst[1] = crockfordAlphabet[((src[0]&0b00000111)<<2)|((src[1]&0b11000000)>>6)]
	dst[2] = crockfordAlphabet[(src[1]&0b00111110)>>1]
	dst[3] = crockfordAlphabet[((src[1]&0b00000001)<<4)|((src[2]&0b11110000)>>4)]
	dst[4] = crockfordAlphabet[((src[2]&0b00001111)<<1)|((src[3]&0b10000000)>>7)]
	dst[5] = crockfordAlphabet[(src[3]&0b01111100)>>2]
	dst[6] = crockfordAlphabet[((src[3]&0b00000011)<<3)|((src[4]&0b11100000)>>5)]
	dst[7] = crockfordAlphabet[src[4]&0b00011111]
	dst[8] = crockfordAlphabet[(src[5]&0b11111000)>>3]
	dst[9] = crockfordAlphabet[((src[5]&0b00000111)<<2)|((src[6]&0b11000000)>>6)]

	// Encode remaining random bytes
	dst[10] = crockfordAlphabet[(src[6]&0b00111110)>>1]
	dst[11] = crockfordAlphabet[((src[6]&0b00000001)<<4)|((src[7]&0b11110000)>>4)]
	dst[12] = crockfordAlphabet[((src[7]&0b00001111)<<1)|((src[8]&0b10000000)>>7)]
	dst[13] = crockfordAlphabet[(src[8]&0b01111100)>>2]
	dst[14] = crockfordAlphabet[((src[8]&0b00000011)<<3)|((src[9]&0b11100000)>>5)]
	dst[15] = crockfordAlphabet[src[9]&0b00011111]
	dst[16] = crockfordAlphabet[(src[10]&0b11111000)>>3]
	dst[17] = crockfordAlphabet[((src[10]&0b00000111)<<2)|((src[11]&0b11000000)>>6)]
	dst[18] = crockfordAlphabet[(src[11]&0b00111110)>>1]
	dst[19] = crockfordAlphabet[((src[11]&0b00000001)<<4)|((src[12]&0b11110000)>>4)]
	dst[20] = crockfordAlphabet[((src[12]&0b00001111)<<1)|((src[13]&0b10000000)>>7)]
	dst[21] = crockfordAlphabet[(src[13]&0b01111100)>>2]
	dst[22] = crockfordAlphabet[((src[13]&0b00000011)<<3)|((src[14]&0b11100000)>>5)]
	dst[23] = crockfordAlphabet[src[14]&0b00011111]
	dst[24] = crockfordAlphabet[(src[15]&0b11111000)>>3]
	dst[25] = crockfordAlphabet[((src[15] & 0b00000111) << 2)]

	return string(dst[:])
}
