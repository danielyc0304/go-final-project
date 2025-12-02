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

// RespondJSON 返回 JSON 回應
func RespondJSON(ctx *context.Context, status int, data interface{}) {
	ctx.Output.SetStatus(status)
	ctx.Output.JSON(data, false, false)
}

// RespondError 返回錯誤回應
func RespondError(ctx *context.Context, status int, message string) {
	ctx.Output.SetStatus(status)
	ctx.Output.JSON(map[string]interface{}{
		"success": false,
		"error":   message,
	}, false, false)
}
