package main

import (
	"bytes"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

type OriginalFormatter struct {
}

func (f *OriginalFormatter) Format(e *log.Entry) ([]byte, error) {
	t := time.Now()
	data := bytes.NewBuffer(make([]byte, 0, 128))
	for k, v := range e.Data {
		data.WriteString(fmt.Sprintf("%s=%s", k, v))
	}

	var msg string
	if data.Len() > 0 {
		msg = fmt.Sprintf("[%s] %s (%s)\n", t.Format("2006-01-02 15:04:05"), e.Message, data)
	} else {
		msg = fmt.Sprintf("[%s] %s\n", t.Format("2006-01-02 15:04:05"), e.Message)
	}
	return []byte(msg), nil
}
