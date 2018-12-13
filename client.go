package main

import (
	"net"
	"time"
)

type Client struct {
	SessionID  string
	Username   string
	RemoteAddr net.Addr
	StartedAt  time.Time
}
