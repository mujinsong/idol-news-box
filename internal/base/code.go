package base

// Code 响应码
type Code int

const (
	CodeSuccess       Code = 200
	CodeBadRequest    Code = 400
	CodeUnauthorized  Code = 401
	CodeNotFound      Code = 404
	CodeBusy          Code = 409
	CodeInvalidParams Code = 422
	CodeServerError   Code = 500
)

const (
	MsgSuccess       = "success"
	MsgBadRequest    = "bad request"
	MsgUnauthorized  = "unauthorized"
	MsgNotFound      = "not found"
	MsgBusy          = "任务正在运行中"
	MsgInvalidParams = "invalid params"
	MsgServerError   = "server error"
)
