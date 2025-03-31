// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package proptest1

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"

	grpcProxy "github.com/mwitkow/grpc-proxy/proxy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

const targetMetadataKey = "x-target"

type requestResponseListener func(ctx context.Context, req, resp proto.Message, method string, err error)

type proxy struct {
	port     int
	targets  sync.Map
	mu       sync.Mutex
	server   *grpc.Server
	listener requestResponseListener
}

type proxyOption func(*proxy)

func withRequestResponseListener(listener requestResponseListener) proxyOption {
	return func(p *proxy) {
		p.listener = listener
	}
}

func newProxy(port int, opts ...proxyOption) *proxy {
	p := &proxy{port: port}

	for _, opt := range opts {
		opt(p)
	}

	p.server = grpc.NewServer(
		grpc.UnknownServiceHandler(grpcProxy.TransparentHandler(p.handler)),
	)

	return p
}

func (p *proxy) addTarget(name string, addresses []string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var conns []*grpc.ClientConn
	for _, addr := range addresses {
		newConn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithCodec(grpcProxy.Codec()))
		if err != nil {
			return "", err
		}
		conns = append(conns, newConn)
	}
	p.targets.Store(name, conns)

	return fmt.Sprintf("localhost:%v", p.port), nil
}

func (p *proxy) removeTarget(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conns, ok := p.targets.Load(name); ok {
		for _, conn := range conns.([]*grpc.ClientConn) {
			conn.Close()
			p.targets.Delete(name)
		}
	}
}

func (p *proxy) handler(
	ctx context.Context,
	_ string,
) (context.Context, *grpc.ClientConn, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	target := md.Get(targetMetadataKey)
	if len(target) == 0 {
		return ctx, nil, fmt.Errorf("missing %s metadata", targetMetadataKey)
	}
	if conns, ok := p.targets.Load(target[0]); ok {
		all := conns.([]*grpc.ClientConn)
		conn := all[rand.Intn(len(all))]
		return ctx, conn, nil
	}
	return ctx, nil, fmt.Errorf("backend not found: %s", target[0])
}

func (p *proxy) start() error {
	addr := fmt.Sprintf("0.0.0.0:%d", p.port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	log.Printf("proxy listening on %s", addr)
	return p.server.Serve(lis)
}

func (p *proxy) stop() {
	log.Println("Shutting down proxy...")
	p.server.GracefulStop()
	p.targets.Range(func(_, value interface{}) bool {
		value.(*grpc.ClientConn).Close()
		return true
	})
}
