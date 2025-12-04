package errors

import "fmt"

// ErrorCode 错误码
type ErrorCode int

const (
	// 0: 成功
	Success ErrorCode = 0

	// 1xxx: 客户端错误
	ErrInvalidParams     ErrorCode = 1001 // 参数错误
	ErrUnauthorized      ErrorCode = 1002 // 未授权
	ErrForbidden         ErrorCode = 1003 // 禁止访问
	ErrNotFound          ErrorCode = 1004 // 资源不存在
	ErrConflict          ErrorCode = 1005 // 资源冲突
	ErrTooManyRequests   ErrorCode = 1006 // 请求过多
	ErrInvalidToken      ErrorCode = 1007 // Token无效
	ErrTokenExpired      ErrorCode = 1008 // Token过期
	ErrInvalidCredentials ErrorCode = 1009 // 用户名或密码错误

	// 2xxx: 业务错误
	ErrNodeNotFound      ErrorCode = 2001 // 节点不存在
	ErrNodeOffline       ErrorCode = 2002 // 节点离线
	ErrNodeAlreadyExists ErrorCode = 2003 // 节点已存在
	ErrTaskNotFound      ErrorCode = 2004 // 任务不存在
	ErrTaskRunning       ErrorCode = 2005 // 任务正在运行
	ErrTaskFailed        ErrorCode = 2006 // 任务执行失败
	ErrUserNotFound      ErrorCode = 2007 // 用户不存在
	ErrUserDisabled      ErrorCode = 2008 // 用户已禁用
	ErrUserAlreadyExists ErrorCode = 2009 // 用户已存在

	// 3xxx: 版本管理错误
	ErrVersionNotFound      ErrorCode = 3001 // 版本不存在
	ErrVersionAlreadyExists ErrorCode = 3002 // 版本已存在
	ErrVersionNotReleased   ErrorCode = 3003 // 版本未发布
	ErrVersionDeprecated    ErrorCode = 3004 // 版本已废弃
	ErrInvalidVersion       ErrorCode = 3005 // 版本号无效
	ErrVersionHashMismatch  ErrorCode = 3006 // 版本哈希不匹配
	ErrVersionSignatureInvalid ErrorCode = 3007 // 版本签名无效

	// 5xxx: 服务器内部错误
	ErrInternalServer ErrorCode = 5001 // 服务器内部错误
	ErrDatabase       ErrorCode = 5002 // 数据库错误
	ErrRedis          ErrorCode = 5003 // Redis错误
	ErrGRPC           ErrorCode = 5004 // gRPC错误
	ErrFileOperation  ErrorCode = 5005 // 文件操作错误
)

// APIError API错误
type APIError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"` // 详细错误信息（可选）
}

// Error 实现error接口
func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// New 创建API错误
func New(code ErrorCode, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}

// NewWithDetails 创建带详细信息的API错误
func NewWithDetails(code ErrorCode, message, details string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// Wrap 包装标准错误
func Wrap(code ErrorCode, message string, err error) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Details: err.Error(),
	}
}

// 预定义的错误
var (
	// 客户端错误
	ErrInvalidParamsMsg     = New(ErrInvalidParams, "参数错误")
	ErrUnauthorizedMsg      = New(ErrUnauthorized, "未授权")
	ErrForbiddenMsg         = New(ErrForbidden, "禁止访问")
	ErrNotFoundMsg          = New(ErrNotFound, "资源不存在")
	ErrConflictMsg          = New(ErrConflict, "资源冲突")
	ErrTooManyRequestsMsg   = New(ErrTooManyRequests, "请求过多")
	ErrInvalidTokenMsg      = New(ErrInvalidToken, "Token无效")
	ErrTokenExpiredMsg      = New(ErrTokenExpired, "Token已过期")
	ErrInvalidCredentialsMsg = New(ErrInvalidCredentials, "用户名或密码错误")

	// 业务错误
	ErrNodeNotFoundMsg      = New(ErrNodeNotFound, "节点不存在")
	ErrNodeOfflineMsg       = New(ErrNodeOffline, "节点离线")
	ErrNodeAlreadyExistsMsg = New(ErrNodeAlreadyExists, "节点已存在")
	ErrTaskNotFoundMsg      = New(ErrTaskNotFound, "任务不存在")
	ErrTaskRunningMsg       = New(ErrTaskRunning, "任务正在运行")
	ErrTaskFailedMsg        = New(ErrTaskFailed, "任务执行失败")
	ErrUserNotFoundMsg      = New(ErrUserNotFound, "用户不存在")
	ErrUserDisabledMsg      = New(ErrUserDisabled, "用户已禁用")
	ErrUserAlreadyExistsMsg = New(ErrUserAlreadyExists, "用户已存在")

	// 版本管理错误
	ErrVersionNotFoundMsg      = New(ErrVersionNotFound, "版本不存在")
	ErrVersionAlreadyExistsMsg = New(ErrVersionAlreadyExists, "版本已存在")
	ErrVersionNotReleasedMsg   = New(ErrVersionNotReleased, "版本未发布")
	ErrVersionDeprecatedMsg    = New(ErrVersionDeprecated, "版本已废弃")
	ErrInvalidVersionMsg       = New(ErrInvalidVersion, "版本号无效")
	ErrVersionHashMismatchMsg  = New(ErrVersionHashMismatch, "版本哈希不匹配")
	ErrVersionSignatureInvalidMsg = New(ErrVersionSignatureInvalid, "版本签名无效")

	// 服务器错误
	ErrInternalServerMsg = New(ErrInternalServer, "服务器内部错误")
	ErrDatabaseMsg       = New(ErrDatabase, "数据库错误")
	ErrRedisMsg          = New(ErrRedis, "Redis错误")
	ErrGRPCMsg           = New(ErrGRPC, "gRPC错误")
	ErrFileOperationMsg  = New(ErrFileOperation, "文件操作错误")
)

// GetHTTPStatus 获取HTTP状态码
func (e *APIError) GetHTTPStatus() int {
	switch {
	case e.Code >= 1000 && e.Code < 2000:
		// 客户端错误
		switch e.Code {
		case ErrUnauthorized, ErrInvalidToken, ErrTokenExpired, ErrInvalidCredentials:
			return 401
		case ErrForbidden:
			return 403
		case ErrNotFound:
			return 404
		case ErrConflict:
			return 409
		case ErrTooManyRequests:
			return 429
		default:
			return 400
		}
	case e.Code >= 2000 && e.Code < 4000:
		// 业务错误
		switch e.Code {
		case ErrNodeNotFound, ErrTaskNotFound, ErrUserNotFound, ErrVersionNotFound:
			return 404
		case ErrNodeAlreadyExists, ErrUserAlreadyExists, ErrVersionAlreadyExists:
			return 409
		case ErrUserDisabled, ErrNodeOffline:
			return 403
		default:
			return 400
		}
	case e.Code >= 5000:
		// 服务器错误
		return 500
	default:
		return 500
	}
}
