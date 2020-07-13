package uuid

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
)

type (
	Format int
	// UUID is an entirely random identifier created by the standard PRNG
	UUID [16]byte
)

const (
	FORMAT_UNKNOWN Format = iota
	FORMAT_LONG
	FORMAT_SHORT
)

// New generates a new UUID
func New() *UUID {
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		panic(fmt.Errorf("error generating uuid: %v", err))
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return b2u(uuid)
}

// NewFromBytes converts a byte array to an UUID
func NewFromBytes(bytes []byte) *UUID {
	return b2u(bytes)
}

func GetFormat(value string) Format {
	const (
		ValidLongChars  = "0123456789abcdef-"
		ValidShortChars = "abcdefghijklmnopqrstuvwxyz0123456789"
	)
	goto Short
Short: // step #1: do we have a short UUID?
	if len(value) == 26 {
		for i := range value {
			if strings.Index(ValidShortChars, string(value[i])) == -1 {
				goto Long
			}
		}
		return FORMAT_SHORT
	}
Long: // step #2: do we have a long UUID?
	if len(value) == 36 {
		for i := range value {
			if strings.Index(ValidLongChars, string(value[i])) == -1 {
				goto None
			}
		}
		return FORMAT_LONG
	}
None: // step #3: not a UUID
	return FORMAT_UNKNOWN
}

func u2b(u *UUID) []byte {
	b := make([]byte, len(u))
	for i := 0; i < len(b); i++ {
		b[i] = u[i]
	}
	return b
}

func b2u(b []byte) *UUID {
	u := new(UUID)
	for i := 0; i < len(b); i++ {
		u[i] = b[i]
	}
	return u
}

// Bytes converts UUID to a byte array
func (u *UUID) Bytes() []byte {
	return u2b(u)
}

// Long formats UUID as a sequence of hex digits separated into 5 groups
func (u *UUID) Long() string {
	return u.LongEx("-")
}

func (u *UUID) LongEx(sep string) string {
	return fmt.Sprintf("%X%s%X%s%X%s%X%s%X", u[0:4], sep, u[4:6], sep, u[6:8], sep, u[8:10], sep, u[10:])
}

// Short converts UUID to lowercase base32 representation without padding (aka 1Password client-side UUIDs)
func (u *UUID) Short() string {
	return strings.ToLower(strings.TrimRight(base32.StdEncoding.EncodeToString(u2b(u)), "="))
}
