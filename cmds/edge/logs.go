package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	. "github.com/ayushmantripathy/url-shortner/internal"
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

type BatchingService struct {
	mu           sync.Mutex
	logs         bytes.Buffer
	flushPending int32
}

func (ser *BatchingService) init(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go ser.flush(ticker)
}

func (ser *BatchingService) flush(ticker *time.Ticker) {
	for {
		<-ticker.C

		if ser.flushPending <= 0 {
			continue
		}

		ser.mu.Lock()
		clone := make([]byte, ser.logs.Len())
		copy(clone, ser.logs.Bytes())
		ser.logs.Reset()
		ser.flushPending = 0
		ser.mu.Unlock()

		natsConn.Emit("logs.redirects", clone)
	}
}

func (ser *BatchingService) pushRedirectLog(r *http.Request, alias string) {
	bytes, _ := json.Marshal(RedirectLog{
		Ip:        readUserIP(r),
		Edge:      instanceName,
		UserAgent: r.UserAgent(),
		Code:      alias,
	})

	ser.mu.Lock()
	defer ser.mu.Unlock()

	ser.flushPending++
	ser.logs.Write(bytes)
	ser.logs.WriteByte('\n')
}
