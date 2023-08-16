package mgp

import (
	"crypto/md5" //nolint:gosec
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/babybabylong/first-business/chargechannel"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

const (
	// orderNoMaxLength 订单号最大长度
	orderNoMaxLength = 36
	// objectIDLength objectID 长度，12个字节，一个字节2个字符
	objectIDLength  = 24
	amountPrecision = 2
)

type ChannelType string // 通道类型

const (
	ChannelTypeEcuador ChannelType = "0"
)

const (
	success    = "1"
	fail       = "2"
	processing = "0"
)

const (
	version  = `V2`
	signType = `MD5`
)

// payArgument 支付接口的参数
type payArgument struct {
	Version     string          `json:"version"`     // 版本号
	SignType    string          `json:"signType"`    // 签名类型
	MerchantNo  string          `json:"merchantNo"`  // 商户号
	Date        string          `json:"date"`        // 时间戳,使用厄瓜多尔时区
	ChannelType string          `json:"channleType"` // 通道类型
	Sign        string          `json:"sign"`        // 加密串
	NoticeURL   string          `json:"noticeUrl"`   // 异步回调地址
	OrderNo     string          `json:"orderNo"`     // 订单号,36字符内
	Amount      decimal.Decimal `json:"bizAmt"`      // 金额
}

func newPayArgument(merchantNo, noticeURL, orderNo string, amount decimal.Decimal, channelType ChannelType) *payArgument {
	location, _ := time.LoadLocation("America/Guayaquil")

	return &payArgument{
		Version:     version,
		SignType:    signType,
		MerchantNo:  merchantNo,
		Date:        time.Now().In(location).Format(`20060102150405`),
		NoticeURL:   noticeURL,
		ChannelType: string(channelType),
		OrderNo:     orderNo,
		Amount:      amount,
	}
}

func (p payArgument) values() url.Values {
	result := url.Values{}
	result.Set(`version`, p.Version)
	result.Set(`signType`, p.SignType)
	result.Set(`merchantNo`, p.MerchantNo)
	result.Set(`date`, p.Date)
	result.Set(`channleType`, p.ChannelType) // 注意:这里是对方接口的拼写错误，与代码无关
	result.Set(`noticeUrl`, p.NoticeURL)
	result.Set(`orderNo`, p.OrderNo)
	result.Set(`bizAmt`, p.Amount.String())

	return result
}
func (p payArgument) sign(privateKey string) string {
	value := p.values()

	parameters := make([]parameter, 0, len(value))

	for key := range value {
		parameters = append(parameters, parameter{
			key:   key,
			value: value.Get(key),
		})
	}

	sort.Slice(parameters, func(i, j int) bool {
		return parameters[i].key < parameters[j].key
	})

	parameters[len(parameters)-1].value += privateKey

	result := make([]string, 0, len(parameters))

	for _, parameter := range parameters {
		result = append(result, parameter.Encode())
	}

	combinedStr := strings.Join(result, `&`)

	println(`combinedStr`, combinedStr)

	return fmt.Sprintf("%x", md5.Sum([]byte(combinedStr))) //nolint:gosec
}

type parameter struct {
	key   string
	value string
}

func (p parameter) Encode() string {
	return fmt.Sprintf(`%s=%s`, p.key, p.value)
}

type payResponse struct {
	Code   string            `json:"code"`           // 响应码， 0成功，1失败
	Msg    string            `json:"msg"`            // 错误信息
	Detail payResponseDetail `json:"detail"`         // 详情
	Info   payResponseInfo   `json:"checkstandInfo"` // 其他信息
}

func (p payResponse) Validate() error {
	if p.Code == `0` {
		return nil
	}

	return errors.New(p.Msg)
}

type payResponseDetail struct {
	PayHTML string `json:"PayHtml"` // 表单格式
	PayURL  string `json:"PayURL"`  // url
}

type payResponseInfo struct {
	QRCodeURL   string `json:"qrcode_url"`      // 二维码地址
	BankAccount string `json:"bankAccount"`     // 银行卡号
	BankCode    string `json:"bankCode"`        // 银行编码
	BankName    string `json:"bankName"`        // 银行名称
	Holder      string `json:"bankAccountName"` // 持卡人
	Money       string `json:"money"`           // 余额
}

// payAsyncResponse 支付异步回调
type payAsyncResponse struct {
	OrderNo    string          `json:"orderNo"`  // 商户订单号
	Amount     decimal.Decimal `json:"orderAmt"` // 订单金额
	PayAmount  decimal.Decimal `json:"bizAmt"`   // 支付金额
	PaidStatus string          `json:"status"`   // 状态，1 成功，2失败，0处理中
	Remark     string          `json:"remark"`   // 状态信息
	Version    string          `json:"version"`  // 常量V2
	Date       string          `json:"date"`     // 时间戳 yyyyMMddHHmmss
	Notes      string          `json:"notes"`    // 订单传入，原值返回
	Sign       string          `json:"sign"`     // 签名
}

func (p payAsyncResponse) RealPayAmount() decimal.Decimal {
	return p.PayAmount
}

func (p payAsyncResponse) Validate(privateKey string) error {
	if p.sign(privateKey) != p.Sign {
		return errors.New("签名错误")
	}

	return nil
}

func (p payAsyncResponse) values() url.Values {
	result := url.Values{}
	result.Set(`orderNo`, p.OrderNo)
	result.Set(`orderAmt`, p.Amount.StringFixed(2))
	result.Set(`bizAmt`, p.PayAmount.StringFixed(2))
	result.Set(`status`, p.PaidStatus)
	result.Set(`remark`, p.Remark)
	result.Set(`version`, p.Version)
	result.Set(`date`, p.Date)
	if p.Notes != "" {
		result.Set(`notes`, p.Notes)
	}

	return result
}

func (p payAsyncResponse) sign(privateKey string) string {
	value := p.values()

	parameters := make([]parameter, 0, len(value))

	for key := range value {
		parameters = append(parameters, parameter{
			key:   key,
			value: value.Get(key),
		})
	}

	sort.Slice(parameters, func(i, j int) bool {
		return parameters[i].key < parameters[j].key
	})

	parameters[len(parameters)-1].value += privateKey

	result := make([]string, 0, len(parameters))

	for _, parameter := range parameters {
		result = append(result, parameter.Encode())
	}

	combinedStr := strings.Join(result, `&`)

	println(`combinedStr`, combinedStr)

	return fmt.Sprintf("%x", md5.Sum([]byte(combinedStr))) //nolint:gosec
}

func (p payAsyncResponse) New() chargechannel.AsyncCallBackTemplate {
	return &payAsyncResponse{}
}
func (p payAsyncResponse) Status() chargechannel.PaidStatus {
	switch p.PaidStatus {
	case success:
		return chargechannel.Paid
	case fail:
		return chargechannel.PaidFail
	case processing:
		return chargechannel.PaidProcessing
	default:
		return chargechannel.PaidUnknown
	}
}

func (p payAsyncResponse) Result() io.Reader {
	if p.Status() == chargechannel.Paid || p.Status() == chargechannel.PaidFail {
		return strings.NewReader(`SUCCESS`)
	}

	return strings.NewReader(``)
}
