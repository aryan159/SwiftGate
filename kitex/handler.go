package main

import (
	"context"
	"fmt"
	api "kitex/kitex_gen/api"
)

// BankServiceImpl implements the last service interface defined in the IDL.
type BankServiceImpl struct{}

// GetNameMethod implements the BankServiceImpl interface.
func (s *BankServiceImpl) GetNameMethod(ctx context.Context, request *api.BankNameReq) (resp *api.BankNameResp, err error) {
	// TODO: Your code here...

	fmt.Println("[Kitex] Request Received")
	fmt.Print("[Kitex] Request: ")
	fmt.Println(request)

	resp = &api.BankNameResp{RespBody: request.Name + " Bank"}

	fmt.Println("[Kitex] Response Generated")
	fmt.Print("[Kitex] Response: ")
	fmt.Println(resp)
	fmt.Print("[Kitex] Returning Response now")

	return resp, nil
}
