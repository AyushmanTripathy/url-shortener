package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	. "github.com/ayushmantripathy/url-shortner/internal"
	"github.com/nats-io/nats.go/jetstream"
)

func handleRedirectLogs(msg jetstream.Msg) {
	defer msg.Ack()

	var logEntry RedirectLog
	var logHeader RedirectLogHeader

	reader := bytes.NewReader(msg.Data())
	scanner := bufio.NewReader(reader)

	var sb strings.Builder
	sb.WriteString("INSERT INTO logs (code, ip, edge, user_agent) VALUES ")

	headerLine, err := scanner.ReadBytes('\n')
	if err != nil {
		log.Println("No header found with logs, ignoring")
		return
	}

	if err = json.Unmarshal(headerLine, &logHeader); err != nil {
		log.Println("Failed to parse header log json, ignoring", err)
		return
	}

	args := make([]any, 0)

	for i := 0; true; i++ {
		line, err := scanner.ReadBytes('\n')
		if err != nil {
			break
		}

		if err = json.Unmarshal(line, &logEntry); err != nil {
			log.Println("Failed to parse log json", err)
			continue
		}

		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, "($%d,$%d,$%d,$%d)", i*4+1, i*4+2, i*4+3, i*4+4)

		args = append(args, logEntry.Code, logEntry.Ip, logHeader.Edge, logEntry.UserAgent)
	}

	if len(args) <= 0 {
		return
	}

	if _, err := Db.Exec(sb.String(), args...); err != nil {
		log.Println("Failed to push logs to Db. source = ", logHeader.Edge)
	}
}
