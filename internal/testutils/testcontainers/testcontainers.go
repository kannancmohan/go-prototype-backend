package testcontainers_testutils

import (
	"context"
	"fmt"
)

type TContainerAddr struct {
	Host    string
	Port    int
	Address string
}

func newTContainerAddr(host string, port int) TContainerAddr {
	address := fmt.Sprintf("%s:%d", host, port)
	return TContainerAddr{Host: host, Port: port, Address: address}
}

type tContainer[T any] interface {
	Start(context.Context) error
	Stop(context.Context) error
	GetContainer() (T, error)
}
