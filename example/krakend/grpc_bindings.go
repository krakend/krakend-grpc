package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	pb "github.com/devopsfaith/krakend-grpc/example/pb"
)

const (
	serviceFullName = "helloworld.Greeter.SayHello"
)

type Registerer string

func (Registerer) RegisterClients(f func(
	name string,
	enc func(context.Context, interface{}) (request interface{}, err error),
	dec func(context.Context, interface{}) (request interface{}, err error),
	t interface{},
)) {
	f(serviceFullName, greetingReqEncoder, greetingRespDecoder, new(pb.HelloReply))
}

type RequestWrapper interface {
	Params() map[string]string
	Header() map[string][]string
	Body() io.Reader
}

func greetingReqEncoder(ctx context.Context, request interface{}) (interface{}, error) {
	req, ok := request.(RequestWrapper)
	if !ok {
		return nil, fmt.Errorf("the received request body has an unexpected type: %T", request)
	}

	t := &pb.HelloRequest{}
	if err := json.NewDecoder(req.Body()).Decode(t); err != nil {
		return nil, err
	}

	return t, nil
}

func greetingRespDecoder(ctx context.Context, grpcReply interface{}) (interface{}, error) {
	reply, ok := grpcReply.(*pb.HelloReply)
	if !ok {
		return responseWrapper{}, fmt.Errorf("the received gRPC reply has an unexpected type: %T", grpcReply)
	}

	return &responseWrapper{
		// TODO: improve the mapping
		d: map[string]interface{}{"content": reply},
		h: map[string][]string{},
	}, nil
}

type responseWrapper struct {
	d map[string]interface{}
	h map[string][]string
}

func (r *responseWrapper) Data() map[string]interface{} {
	return r.d
}

func (r *responseWrapper) Header() map[string][]string {
	return r.h
}
