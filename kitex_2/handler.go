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

	resp.RespBody = fmt.Sprintf("%v Bank Bank", request.Name)

	return resp, nil
}
