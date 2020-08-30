package main

import (
	"fmt"
	"net"
	"sync"
)

type serverOptions struct {
	maxConcurrentStreams  uint32
	maxReceiveMessageSize int
	maxSendMessageSize    int
}

type Server struct {
	opts serverOptions

	mu     sync.Mutex // guards following
	lis    map[net.Listener]bool
}

var defaultOptions = serverOptions{
	maxConcurrentStreams:  100,
	maxReceiveMessageSize: 100,
	maxSendMessageSize:    100,
}

type serverOption interface {
	apply(*serverOptions)
}

type maxRecvMsgSize struct {
	size int
}

func (MaxRecv *maxRecvMsgSize) apply(opt *serverOptions) {
	opt.maxReceiveMessageSize =  MaxRecv.size
}

type MaxSendMsgSize struct {
	size int
}

func (MaxSend *MaxSendMsgSize) apply(opt *serverOptions) {
	opt.maxSendMessageSize = MaxSend.size
}

func NewServer(opt ...serverOption) *Server {
	opts := defaultOptions
	for _, o := range opt {
		o.apply(&opts)
	}

	fmt.Println(opts)

	return &Server{
		opts: opts,
		mu:   sync.Mutex{},
		lis:  nil,
	}
}

func main() {
	maxRecvMsgSize := maxRecvMsgSize{size: 123}
	MaxSendMsgSize := MaxSendMsgSize{size: 456}
	NewServer(&maxRecvMsgSize, &MaxSendMsgSize)
}
