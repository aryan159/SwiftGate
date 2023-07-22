package main

import (
	api "bank/kitex_gen/api"
	"fmt"
	"context"
)

// BankImpl implements the last service interface defined in the IDL.
type BankImpl struct{}

// Name implements the BankImpl interface.
func (s *BankImpl) Name(ctx context.Context, request *api.BankNameReq) (resp *api.BankNameResp, err error) {
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
