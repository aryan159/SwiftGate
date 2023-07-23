package main

import (
	api "bank/kitex_gen/api"
	"context"
	"errors"
	"fmt"
	"math/rand"
)

// BankImpl implements the last service interface defined in the IDL.
type BankImpl struct{}

// Name implements the BankImpl interface.
func (s *BankImpl) Name(ctx context.Context, request *api.BankNameReq) (resp *api.BankNameResp, err error) {

	fmt.Println("[Kitex] Request Received")
	fmt.Print("[Kitex] Request: ")
	fmt.Println(request)

	// Randomly Error out
	r := rand.Intn(100)
	if r < 75 {
		fmt.Println("[Kitex] Random error triggered!")
		return nil, errors.New("50% Error")
	}

	resp = &api.BankNameResp{RespBody: request.Name + " Bank"}

	fmt.Println("[Kitex] Response Generated")
	fmt.Print("[Kitex] Response: ")
	fmt.Println(resp)
	fmt.Print("[Kitex] Returning Response now")

	return resp, nil
}
