package grpc

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/logging"
	"github.com/devopsfaith/krakend/proxy"
	grpcKit "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	Namespace      = "github.com/devopsfaith/krakend-grpc"
	EncodingPrefix = "grpc-"
)

// NewGRPCProxy is a proxy.BackendFactory for consuming gRPC services
func NewGRPCProxy(l logging.Logger, f proxy.BackendFactory) proxy.BackendFactory {
	return func(remote *config.Backend) proxy.Proxy {
		bo := getOptions(remote)

		if bo == nil {
			l.Debug("gRPC: client factory not used for", remote)
			return f(remote)
		}

		next := grpcKit.NewClient(
			newConnection(l, remote),
			bo.serviceName,
			bo.method,
			bo.enc,
			bo.dec,
			bo.grpcReply,
			bo.options...,
		).Endpoint()

		return func(ctx context.Context, request *proxy.Request) (*proxy.Response, error) {
			req := &requestWrapper{
				params: request.Params,
				header: request.Headers,
				body:   request.Body,
			}

			resp, err := next(ctx, req)
			request.Body.Close()
			if err != nil {
				l.Warning("gRPC calling the next mw:", err.Error())
				return nil, err
			}

			r, ok := resp.(ResponseWrapper)
			if !ok {
				err := fmt.Errorf("gRPC: the received response has an unexpected type: %T", resp)
				l.Warning("gRPC casting the response:", err.Error())
				return nil, err
			}
			return &proxy.Response{
				Data:       r.Data(),
				IsComplete: true,
				Metadata: proxy.Metadata{
					Headers: r.Header(),
				},
			}, nil
		}
	}
}

type RequestWrapper interface {
	Params() map[string]string
	Header() map[string][]string
	Body() io.Reader
}

type ResponseWrapper interface {
	Data() map[string]interface{}
	Header() map[string][]string
}

func RegisterClient(
	name string,
	enc func(context.Context, interface{}) (request interface{}, err error),
	dec func(context.Context, interface{}) (request interface{}, err error),
	t interface{},
) {
	clientRegistry.Set(name, enc, dec, t)
}

type requestWrapper struct {
	params map[string]string
	header map[string][]string
	body   io.Reader
}

func (r *requestWrapper) Params() map[string]string   { return r.params }
func (r *requestWrapper) Header() map[string][]string { return r.header }
func (r *requestWrapper) Body() io.Reader             { return r.body }

func newConnection(l logging.Logger, remote *config.Backend) *grpc.ClientConn {
	var backendCertPath string
	if e, ok := remote.ExtraConfig[Namespace]; ok {
		if s, ok := e.(string); ok {
			backendCertPath = s
		}
	}

	var do grpc.DialOption
	if backendCertPath != "" {
		cred, err := credentials.NewClientTLSFromFile(backendCertPath, "")
		if err != nil {
			l.Fatal("gRPC unable to get credentials: " + err.Error())
		}
		do = grpc.WithTransportCredentials(cred)
	} else {
		l.Debug("gRPC: connecting to ", remote.Host[0], " with an insecure connection")
		do = grpc.WithInsecure()
	}
	cc, err := grpc.Dial(remote.Host[0], do)
	if err != nil {
		l.Fatal("gRPC unable to Dial: " + err.Error())
	}
	return cc
}

type client struct {
	Encode    func(context.Context, interface{}) (request interface{}, err error)
	Decode    func(context.Context, interface{}) (response interface{}, err error)
	ReplyType interface{}
}

var clientRegistry = registry{
	r:  map[string]client{},
	mu: new(sync.RWMutex),
}

type registry struct {
	r  map[string]client
	mu *sync.RWMutex
}

func (r *registry) Set(
	name string,
	enc func(context.Context, interface{}) (request interface{}, err error),
	dec func(context.Context, interface{}) (request interface{}, err error),
	t interface{},
) {
	r.mu.Lock()
	r.r[name] = client{
		Encode:    enc,
		Decode:    dec,
		ReplyType: t,
	}
	r.mu.Unlock()
}

func (r *registry) Get(name string) (client, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	v, ok := r.r[name]
	return v, ok
}

type backendOptions struct {
	serviceName string
	method      string
	enc         grpcKit.EncodeRequestFunc
	dec         grpcKit.DecodeResponseFunc
	grpcReply   interface{}
	options     []grpcKit.ClientOption
}

func getOptions(remote *config.Backend) *backendOptions {
	if !strings.HasPrefix(remote.Encoding, EncodingPrefix) {
		return nil
	}

	c, ok := clientRegistry.Get(remote.Encoding[len(EncodingPrefix):])
	if !ok {
		return nil
	}
	return &backendOptions{
		method:      remote.Method,
		serviceName: strings.TrimPrefix(remote.URLPattern, "/"),
		enc:         c.Encode,
		dec:         c.Decode,
		grpcReply:   c.ReplyType,
	}
}
