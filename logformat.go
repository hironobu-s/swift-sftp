package main

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

type SftpLogFormatter struct {
}

func (f *SftpLogFormatter) Format(e *logrus.Entry) ([]byte, error) {
	t := time.Now()

	// client
	var client *Client
	data, ok := e.Data["client"]
	if ok {
		tmp, ok := data.(*Client)
		if ok {
			client = tmp
		}
	}

	var msg string
	if client != nil {
		// we need to shorten session id because it's too long to display
		msg = fmt.Sprintf("%s [%s]  %s\n",
			t.Format("2006-01-02 15:04:05"),
			client.SessionID[:16],
			e.Message)

	} else {
		msg = fmt.Sprintf("%s [%s]  %s\n",
			t.Format("2006-01-02 15:04:05"),
			"-",
			e.Message)
	}
	return []byte(msg), nil
}
