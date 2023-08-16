package chargechannel

import (
	"context"
	"io"

	"github.com/shopspring/decimal"
)

// Channel 充值渠道
type Channel interface {
	// Key 充值渠道key
	Key() ChannelKey
	PrivateKey() string
	// NeedCheck 是否需要主动查单,当need==false时，template!=nil,表示是一个异步查单(主动回调)的返回值模板
	NeedCheck() (template AsyncCallBackTemplate, need bool)
	// CreateOrderNo 返回一个商户订单号(每个渠道的规则不同,由实现生成)
	CreateOrderNo(id int64, amount decimal.Decimal) string
	// CreateOrder 创建订单，分别是商户订单号，金额，回调地址
	CreateOrder(ctx context.Context, orderNo string, amount decimal.Decimal, callbackURL string, extend *CreateOrderExtendParam) (payUrl, payHtml string, err error)
	// Check 查单
	Check(channelOrderNo string) (paid PaidStatus, err error)
}

// CreateOrderExtendParam 创建订单额外参数
type CreateOrderExtendParam struct { // 注意：这个参数目前只有shop-club的EPay支付渠道有效
	PayCode    int    // ChannelKeyEPayRuble
	UserID     int64  // 用户ID
	UserIP     string // 用户IP
	SuccessURL string // 成功后跳转的URL
}

// Manager 充值渠道管理器
type Manager interface {
	// Register 注册渠道
	Register(channel Channel) error
	// LoadByKey 通过key加载渠道
	LoadByKey(key ChannelKey) (channel Channel, err error)
	// LoadTemplateBy 通过key加载回调模板
	LoadTemplateBy(key ChannelKey) (template AsyncCallBackTemplate, err error)
}

// AsyncCallBackTemplate 异步回调接口
type AsyncCallBackTemplate interface {
	// New 新建
	New() AsyncCallBackTemplate
	// Validate 校验,判断数据是否正确
	Validate(privateKey string) error
	// Status 支付状态
	Status() PaidStatus
	// Result 是返回给异步回调接口的body
	Result() io.Reader
	// RealPayAmount 真实支付金额
	RealPayAmount() decimal.Decimal
}

type Accessor interface {
	// SetRecordStarted 设置订单下单情况
	SetRecordStarted(id int64, orderNo string, err error) error
	SetRecordFinish(key ChannelKey, orderNo string, realAmount decimal.Decimal, err error) error
}
