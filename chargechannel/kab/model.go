package kab

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/babybabylong/first-business/chargechannel"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"io"
	"strings"
)

const (
	// orderNoMaxLength 订单号最大长度
	orderNoMaxLength = 36
	// objectIDLength objectID 长度，12个字节，一个字节2个字符
	objectIDLength  = 24
	amountPrecision = 2
)

type chargeResponse struct {
	Status  int                `json:"status"`
	Message string             `json:"message"`
	Data    chargeResponseData `json:"data"`
}

type chargeResponseData struct {
	OrderNo string          `json:"order_no"`
	TradeNo string          `json:"trade_no"`
	Amount  decimal.Decimal `json:"amount"`
	Qrcode  string          `json:"qrcode"`
	PayURL  string          `json:"pay_url"`
}

func (c chargeResponse) Validate() error {
	if c.Status == 1 {
		return nil
	}

	return errors.New(c.Message)
}

// PayAsyncResponse 支付异步回调
type payAsyncResponse struct {
	Orderid string          `json:"orderid"`
	Amount  decimal.Decimal `json:"amount"`
	PayNo   string          `json:"payno"`
	Sign    string          `json:"sign"`
}

func (p payAsyncResponse) New() chargechannel.AsyncCallBackTemplate {
	return &payAsyncResponse{}
}

func (p payAsyncResponse) Validate(privateKey string) error {
	if p.sign(privateKey) != p.Sign {
		return errors.New("签名错误")
	}

	return nil
}

func (p payAsyncResponse) sign(privateKey string) (result string) {
	return sign(p.Orderid + p.Amount.StringFixed(2) + p.PayNo + privateKey)
}

func (p payAsyncResponse) Status() chargechannel.PaidStatus {
	return chargechannel.Paid
}

func (p payAsyncResponse) Result() io.Reader {
	return strings.NewReader(`ok`)
}

func (p payAsyncResponse) RealPayAmount() decimal.Decimal {
	return p.Amount.Div(decimal.NewFromInt(100)) // 单位为分
}

func sign(source string) (result string) {
	h := md5.New()
	h.Write([]byte(source))

	return hex.EncodeToString(h.Sum(nil))
}
