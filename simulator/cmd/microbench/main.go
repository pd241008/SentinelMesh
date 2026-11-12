package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"time"
)

// 32-byte wire struct layout:
// - 2 bytes: node_id (uint16)
// - 2 bytes: category bitmask (uint16)
// - 6 bytes: 3 x float16 scores (or 3 x uint16 fixed-point)
// - 6 bytes: timestamp (48-bit millisecond epoch)
// - 16 bytes: truncated HMAC-SHA256 signature
type WireDigest [32]byte

func benchmarkHMACSign(key []byte, data []byte, iters int) time.Duration {
	start := time.Now()
	for i := 0; i < iters; i++ {
		mac := hmac.New(sha256.New, key)
		mac.Write(data)
		sig := mac.Sum(nil)[:16]
		_ = sig
	}
	return time.Since(start) / time.Duration(iters)
}

func benchmarkHMACVerify(key []byte, data []byte, sig []byte, iters int) time.Duration {
	start := time.Now()
	for i := 0; i < iters; i++ {
		mac := hmac.New(sha256.New, key)
		mac.Write(data)
		expected := mac.Sum(nil)[:16]
		_ = hmac.Equal(sig, expected)
	}
	return time.Since(start) / time.Duration(iters)
}

func benchmarkWirePack(iters int) time.Duration {
	start := time.Now()
	var buf WireDigest
	for i := 0; i < iters; i++ {
		binary.BigEndian.PutUint16(buf[0:2], uint16(42))
		binary.BigEndian.PutUint16(buf[2:4], uint16(3))
		binary.BigEndian.PutUint16(buf[4:6], uint16(math.Float32bits(1.234)>>16))
		binary.BigEndian.PutUint16(buf[6:8], uint16(math.Float32bits(0.852)>>16))
		binary.BigEndian.PutUint16(buf[8:10], uint16(math.Float32bits(2.115)>>16))
		// 6 bytes timestamp
		now := time.Now().UnixMilli()
		buf[10] = byte(now >> 40)
		buf[11] = byte(now >> 32)
		buf[12] = byte(now >> 24)
		buf[13] = byte(now >> 16)
		buf[14] = byte(now >> 8)
		buf[15] = byte(now)
	}
	return time.Since(start) / time.Duration(iters)
}

func benchmarkScoringBatch(batchSize int, iters int) time.Duration {
	// Dummy features: dur, sbytes, dbytes, ct_srv_dst
	start := time.Now()
	for it := 0; it < iters; it++ {
		maxDos := 0.0
		maxRecon := 0.0
		for i := 0; i < batchSize; i++ {
			sbytes := float64(1500 + i*10)
			dur := float64(0.01 + float64(i)*0.001)
			vol := sbytes / dur
			if vol > maxDos {
				maxDos = vol
			}
			ct := float64(5 + i%20)
			rec := 1.0 / (ct + 1.0)
			if rec > maxRecon {
				maxRecon = rec
			}
		}
		_ = maxDos
		_ = maxRecon
	}
	return time.Since(start) / time.Duration(iters)
}

func benchmarkUDPLoopback(iters int) (time.Duration, error) {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	conn1, err := net.ListenUDP("udp", addr)
	if err != nil {
		return 0, err
	}
	defer conn1.Close()

	destAddr := conn1.LocalAddr().(*net.UDPAddr)
	conn2, err := net.DialUDP("udp", nil, destAddr)
	if err != nil {
		return 0, err
	}
	defer conn2.Close()

	pkt := make([]byte, 32)
	recvBuf := make([]byte, 64)

	start := time.Now()
	for i := 0; i < iters; i++ {
		_, err := conn2.Write(pkt)
		if err != nil {
			return 0, err
		}
		_, _, err = conn1.ReadFromUDP(recvBuf)
		if err != nil {
			return 0, err
		}
	}
	return time.Since(start) / time.Duration(iters), nil
}

func main() {
	key := []byte("0123456789abcdef0123456789abcdef")
	payload := make([]byte, 16)
	sig := make([]byte, 16)

	const N = 100000

	durScoring := benchmarkScoringBatch(100, N)
	durPack := benchmarkWirePack(N)
	durSign := benchmarkHMACSign(key, payload, N)
	durVerify := benchmarkHMACVerify(key, payload, sig, N)
	durUDP, _ := benchmarkUDPLoopback(10000)

	fmt.Printf("100-flow batch scoring: %v\n", durScoring)
	fmt.Printf("32-byte wire struct packing: %v\n", durPack)
	fmt.Printf("HMAC-SHA256 sign (16-byte tag): %v\n", durSign)
	fmt.Printf("HMAC-SHA256 verify: %v\n", durVerify)
	fmt.Printf("Single UDP loopback roundtrip: %v\n", durUDP)

	totalPerRound := durScoring + durPack + durSign + 3*durVerify
	fmt.Printf("Total per-node per-round CPU time (f=3): %v\n", totalPerRound)
}
