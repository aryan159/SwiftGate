package main

import (
	"context"
	"fmt"
	api "rpc/kitex_gen/api"
)

// EchoImpl implements the last service interface defined in the IDL.
type EchoImpl struct{}

// Echo implements the EchoImpl interface.
func (s *EchoImpl) Echo(ctx context.Context, req *api.Request) (resp *api.Response, err error) {
	fmt.Println("[Kitex] Request Received")
	return &api.Response{Message: "New Message"}, nil
}
