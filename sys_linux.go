//go:build linux

package main

import "syscall"

type sysProcAttrType = syscall.SysProcAttr

func detachedSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
