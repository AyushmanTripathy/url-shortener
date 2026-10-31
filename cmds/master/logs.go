package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"strings"

	. "github.com/ayushmantripathy/url-shortner/internal"
	"github.com/ayushmantripathy/url-shortner/pb"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/proto"
)

func getProtoBufferBytes(payload *[]byte, offset *uint32) ([]byte, bool) {
	payloadLength := uint32(len(*payload))
	tmp := *offset
	if payloadLength-tmp <= 2 {
		return nil, true
	}

	length := uint32(binary.BigEndian.Uint16((*payload)[tmp : tmp+2]))
	if payloadLength-tmp < 2+length {
		return nil, true
	}

	*offset += uint32(length) + 2
	return (*payload)[tmp+2 : tmp+length+2], false
}

func handleRedirectLogs(msg jetstream.Msg) {
	defer msg.Ack()

	var err error
	var offset uint32
	var logEntry pb.RedirectLog
	var logHeader pb.RedirectLogHeader

	payload := msg.Data()

	var sb strings.Builder
	sb.WriteString("INSERT INTO logs (code, ip, edge, user_agent) VALUES ")

	if headerBytes, ended := getProtoBufferBytes(&payload, &offset); ended {
		log.Println("Failed to read RedirectLogHeader bytes, ignoring")
		return
	} else if err = proto.Unmarshal(headerBytes, &logHeader); err != nil {
		log.Println("Failed to parse RedirectLogHeader, ignoring", err, headerBytes)
		return
	}

	args := make([]any, 0)

	i := 0
	for {

		if logBytes, ended := getProtoBufferBytes(&payload, &offset); ended {
			break
		} else if err = proto.Unmarshal(logBytes, &logEntry); err != nil {
			log.Println("Failed to parse RedirectLog, ignoring", err)
			continue
		}

		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, "($%d,$%d,$%d,$%d)", i*4+1, i*4+2, i*4+3, i*4+4)

		args = append(args, logEntry.Alias, logEntry.Ip, logHeader.Edge, logEntry.UserAgent)
		logEntry.Reset()
		i++
	}
	if int(logHeader.Count) != i {
		log.Println("Log count mismatch, expected", logHeader.Count, "got", i)
	}

	if len(args) <= 0 {
		return
	}

	if _, err := Db.Exec(sb.String(), args...); err != nil {
		log.Println("Failed to push logs to Db. source = ", logHeader.Edge)
	}
}
