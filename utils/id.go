package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"os"
	"sync/atomic"
	"time"
)

const encodedLen = 20
const encoding = "0123456789abcdefghijklmnopqrstuv"

var machine = make([]byte, 3)
var pid int = os.Getpid()
var counter uint32 = randInt()

func init() {
	hid, _ := os.Hostname()
	hw := sha256.New()
	hw.Write([]byte(hid))
	copy(machine, hw.Sum(nil))
}

func XID() string {
	// Create a 12-byte slice
	id := make([]byte, 12)

	binary.BigEndian.PutUint32(id[0:4], uint32(time.Now().Unix()))

	id[4] = machine[0]
	id[5] = machine[1]
	id[6] = machine[2]

	id[7] = byte(pid >> 8)
	id[8] = byte(pid)

	i := atomic.AddUint32(&counter, 1)
	id[9] = byte(i >> 16)
	id[10] = byte(i >> 8)
	id[11] = byte(i)

	text := make([]byte, encodedLen)
	encode(text, id[:])
	return string(text)
}

func randInt() uint32 {
	b := make([]byte, 3)
	_, _ = rand.Reader.Read(b)
	return uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2])
}

func encode(dst, id []byte) {
	_ = dst[19]
	_ = id[11]

	dst[19] = encoding[(id[11]<<4)&0x1F]
	dst[18] = encoding[(id[11]>>1)&0x1F]
	dst[17] = encoding[(id[11]>>6)|(id[10]<<2)&0x1F]
	dst[16] = encoding[id[10]>>3]
	dst[15] = encoding[id[9]&0x1F]
	dst[14] = encoding[(id[9]>>5)|(id[8]<<3)&0x1F]
	dst[13] = encoding[(id[8]>>2)&0x1F]
	dst[12] = encoding[id[8]>>7|(id[7]<<1)&0x1F]
	dst[11] = encoding[(id[7]>>4)|(id[6]<<4)&0x1F]
	dst[10] = encoding[(id[6]>>1)&0x1F]
	dst[9] = encoding[(id[6]>>6)|(id[5]<<2)&0x1F]
	dst[8] = encoding[id[5]>>3]
	dst[7] = encoding[id[4]&0x1F]
	dst[6] = encoding[id[4]>>5|(id[3]<<3)&0x1F]
	dst[5] = encoding[(id[3]>>2)&0x1F]
	dst[4] = encoding[id[3]>>7|(id[2]<<1)&0x1F]
	dst[3] = encoding[(id[2]>>4)|(id[1]<<4)&0x1F]
	dst[2] = encoding[(id[1]>>1)&0x1F]
	dst[1] = encoding[(id[1]>>6)|(id[0]<<2)&0x1F]
	dst[0] = encoding[id[0]>>3]
}
