package testcontainers_testutils

import (
	"context"
	"fmt"
)

type testContainerAddr struct {
	Host    string
	Port    int
	Address string
}

func newTestContainerAddr(host string, port int) testContainerAddr {
	address := fmt.Sprintf("%s:%d", host, port)
	return testContainerAddr{Host: host, Port: port, Address: address}
}

type tContainer[T any] interface {
	Start(context.Context) error
	Stop(context.Context) error
	GetContainer() (T, error)
}
