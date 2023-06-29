// Code generated by hertz generator.

package api

import (
	"context"

	"github.com/aryan159/SwiftGate/hertz_2/biz/model/api"
	"github.com/aryan159/SwiftGate/kitex_2/kitex_gen/api/bankservice"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/kitex/client"
)

// GetNameMethod .
// @router /bank/name [GET]
func GetNameMethod(ctx context.Context, c *app.RequestContext) {
	var err error
	var req api.BankNameReq
	err = c.BindAndValidate(&req)
	if err != nil {
		c.String(consts.StatusBadRequest, err.Error())
		return
	}

	client, err := bankservice.NewClient("student", client.WithHostPorts("127.0.0.1:8888"))

	// resp := new(api.BankNameResp)
	// resp.RespBody = fmt.Sprintf("%v Bank", req.GetName())

	resp, err := client.QueryStudent(ctx, req)
	if err != nil {
		panic(err)
	}

	c.JSON(consts.StatusOK, resp)
}