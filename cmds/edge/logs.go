package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ayushmantripathy/url-shortner/pb"
	"google.golang.org/protobuf/proto"
)

func readUserIP(r *http.Request) string {
	iPAddress := r.Header.Get("X-Real-Ip")
	if iPAddress == "" {
		iPAddress = r.Header.Get("X-Forwarded-For")
	}
	if iPAddress == "" {
		iPAddress = r.RemoteAddr
	}
	return iPAddress
}

func encodeProtoBufLength(length int) [2]byte {
	var lengthBuf [2]byte
	binary.BigEndian.PutUint16(lengthBuf[:], uint16(length))
	return lengthBuf
}

type BatchingService struct {
	mu           sync.Mutex
	logs         bytes.Buffer
	flushPending uint32
}

func (ser *BatchingService) init(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go ser.flush(ticker)
}

func (ser *BatchingService) flush(ticker *time.Ticker) {
	for {
		<-ticker.C

		ser.mu.Lock()
		if ser.flushPending <= 0 {
			ser.mu.Unlock()
			continue
		}

		clone := make([]byte, ser.logs.Len())
		logCount := ser.flushPending
		copy(clone, ser.logs.Bytes())
		ser.logs.Reset()
		ser.flushPending = 0
		ser.mu.Unlock()

		header, err := proto.Marshal(&pb.RedirectLogHeader{
			Count: logCount,
			Edge:  instanceName,
		})
		if err != nil {
			log.Println("Failed to marshall RedirectLogHeader, giving up", err)
			continue
		}

		lenBuf := encodeProtoBufLength(len(header))
		combined := make([]byte, len(lenBuf)+len(header)+len(clone))

		copy(combined, lenBuf[:])
		copy(combined[len(lenBuf):], header)
		copy(combined[len(header)+len(lenBuf):], clone)

		natsConn.Emit("logs.redirects", combined)
	}
}

func (ser *BatchingService) pushRedirectLog(r *http.Request, alias string) {
	bytes, err := proto.Marshal(&pb.RedirectLog{
		Ip:        readUserIP(r),
		UserAgent: r.UserAgent(),
		Alias:     alias,
	})
	lenBuf := encodeProtoBufLength(len(bytes))
	if err != nil {
		log.Println("Failed to marshal RedirectLog, ignoring.")
		return
	}

	ser.mu.Lock()
	defer ser.mu.Unlock()

	ser.flushPending++
	ser.logs.Write(lenBuf[:])
	ser.logs.Write(bytes)
}
