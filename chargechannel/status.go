package chargechannel

// PaidStatus 支付状态
type PaidStatus int

const (
	// Paid 已支付
	Paid PaidStatus = 1
	// PaidFail 支付失败
	PaidFail PaidStatus = 2
	// PaidProcessing 处理中
	PaidProcessing PaidStatus = 3
	// PaidUnknown 未知，服务端返回了未知的状态码
	PaidUnknown PaidStatus = -1
)
