package utils

import "github.com/beego/beego/v2/server/web/context"

type APIResponse struct {
	Success bool `json:"success"`
	Status  int  `json:"status"`
	Data    any  `json:"data,omitempty"`
	Error   any  `json:"error,omitempty"`
}

func CreateAPIResponse(ctx *context.Context, status int, data any) {
	ctx.Output.SetStatus(status)
	if status >= 200 && status < 300 {
		ctx.Output.JSON(APIResponse{
			Success: true,
			Status:  status,
			Data:    data,
		}, false, false)
	} else {
		ctx.Output.JSON(APIResponse{
			Success: false,
			Status:  status,
			Error:   data,
		}, false, false)
	}
}
