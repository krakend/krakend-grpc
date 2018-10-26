package grpc

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/logging"
	"github.com/devopsfaith/krakend/proxy"
	pb "github.com/kpacha/krakend-grpc/example/pb"
)

func TestClient(t *testing.T) {
	explosiveBackendFactory := func(remote *config.Backend) proxy.Proxy {
		t.Error("this factory should not been called")
		return proxy.NoopProxy
	}
	buf := new(bytes.Buffer)
	l, _ := logging.NewLogger("DEBUG", buf, "[KRAKEND]")
	ClientRegistry = map[string]Client{"helloworld.Greeter.SayHello": greeterClient}

	bf := NewGRPCProxy(l, explosiveBackendFactory)

	defer func() {
		fmt.Println(buf.String())
		ClientRegistry = map[string]Client{}
	}()

	p := bf(&config.Backend{
		Method:     "SayHello",
		URLPattern: "helloworld.Greeter",
		Encoding:   "grpc-helloworld.Greeter.SayHello",
		Host:       []string{"localhost:50051"},
	})

	resp, err := p(context.Background(), &proxy.Request{Params: map[string]string{"name": "supu"}})
	fmt.Println(resp)
	if err != nil {
		t.Error(err.Error())
	}
}

var greeterClient = Client{
	Encode: func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*proxy.Request)
		return &pb.HelloRequest{Name: req.Params["name"]}, nil
	},
	Decode: func(ctx context.Context, grpcReply interface{}) (interface{}, error) {
		reply := grpcReply.(*pb.HelloReply)
		return &proxy.Response{
			Data:       map[string]interface{}{"content": reply},
			IsComplete: true,
		}, nil
	},
	ReplyType: new(pb.HelloReply),
}
