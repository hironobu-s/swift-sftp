package main

import "syscall"

func dummyStat() interface{} {
	return &syscall.Stat_t{Uid: 65534, Gid: 65534}
}
