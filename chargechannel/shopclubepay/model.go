package shopclubepay

import (
	"crypto/md5"
	"fmt"
	"github.com/babybabylong/first-business/chargechannel"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"
)

type payArgument struct {
	MchID        int    `json:"mch_id"`
	PayCode      int    `json:"pay_code"`
	OrderNo      string `json:"order_no"`
	Price        int64  `json:"price"`
	AppID        int    `json:"app_id"`
	UserIp       string `json:"user_ip"`
	UserID       string `json:"user_id"`
	PayNoticeURL string `json:"pay_notice_url"`
	PayJumpURL   string `json:"pay_jump_url"`
	Time         int64  `json:"time"`
	Sign         string `json:"sign"`
}

func newPayArgument(mchID int, payCode int, orderNo string, price int64, appID int, userIp string, userID string, payNoticeURL string, payJumpURL string) *payArgument {
	return &payArgument{MchID: mchID, PayCode: payCode, OrderNo: orderNo, Price: price, AppID: appID, UserIp: userIp, UserID: userID, PayNoticeURL: payNoticeURL, PayJumpURL: payJumpURL, Time: time.Now().Unix()}
}

func (p payArgument) values() url.Values {
	result := url.Values{}

	result.Set("mch_id", fmt.Sprintf("%d", p.MchID))
	result.Set("pay_code", fmt.Sprintf("%d", p.PayCode))
	result.Set("order_no", p.OrderNo)
	result.Set("price", fmt.Sprintf("%d", p.Price))
	result.Set("app_id", fmt.Sprintf("%d", p.AppID))
	result.Set("pay_notice_url", p.PayNoticeURL)
	result.Set("pay_jump_url", p.PayJumpURL)
	result.Set("time", fmt.Sprintf("%d", p.Time))

	return result
}

type parameter struct {
	key   string
	value string
}

func (p parameter) Encode() string {
	return fmt.Sprintf(`%s=%s`, p.key, p.value)
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

	parameters = append(parameters, parameter{
		key:   "key",
		value: privateKey,
	})

	result := make([]string, 0, len(parameters))

	for _, parameter := range parameters {
		result = append(result, parameter.Encode())
	}

	combinedStr := strings.ToUpper(strings.Join(result, `&`))

	println(`combinedStr`, combinedStr)

	return fmt.Sprintf("%x", md5.Sum([]byte(combinedStr))) //nolint:gosec
}

type payResponseCode struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (p payResponseCode) Validate() error {
	if p.Code == 0 {
		return nil
	}

	return errors.New(p.Msg)
}

type payResponse struct {
	payResponseCode
	Data payResponseDetail `json:"data"`
}

type payResponseDetail struct {
	PayURL     string `json:"pay_url"`
	OrderNo    string `json:"order_no"`
	DisOrderNo string `json:"Dis_order_no"`
}

// payAsyncResponse 支付异步回调
type payAsyncResponse struct {
	MchID      int    `json:"mch_id"`       // mch_id
	OrderNo    string `json:"order_no"`     // 订单号
	DisOrderNo string `json:"dis_order_no"` // 支付平台的单号
	RealPrice  int64  `json:"real_price"`   // 实际收到的金额
	OrderPrice int64  `json:"order_price"`  // 订单原金额
	NtiTime    int64  `json:"nti_time"`     // UTC时间戳 秒
	Code       int    `json:"code"`         // 只有1为成功
	Sign       string `json:"sign"`         // 签名
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

func (p payAsyncResponse) values() url.Values {
	result := url.Values{}
	result.Set(`mch_id`, fmt.Sprintf("%d", p.MchID))
	result.Set(`order_no`, p.OrderNo)
	result.Set(`dis_order_no`, p.DisOrderNo)
	result.Set(`real_price`, fmt.Sprintf("%d", p.RealPrice))
	result.Set(`order_price`, fmt.Sprintf("%d", p.OrderPrice))
	result.Set(`nti_time`, fmt.Sprintf("%d", p.NtiTime))
	result.Set(`code`, fmt.Sprintf("%d", p.Code))

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

	parameters = append(parameters, parameter{
		key:   "key",
		value: privateKey,
	})

	result := make([]string, 0, len(parameters))

	for _, parameter := range parameters {
		result = append(result, parameter.Encode())
	}

	combinedStr := strings.ToUpper(strings.Join(result, `&`))

	println(`combinedStr`, combinedStr)

	return fmt.Sprintf("%x", md5.Sum([]byte(combinedStr))) //nolint:gosec
}

func (p payAsyncResponse) Status() chargechannel.PaidStatus {
	switch p.Code {
	case 1:
		return chargechannel.Paid
	default:
		return chargechannel.PaidFail
	}
}

func (p payAsyncResponse) Result() io.Reader {
	if p.Status() == chargechannel.Paid {
		return strings.NewReader(`success`)
	}

	return strings.NewReader(``)
}

func (p payAsyncResponse) RealPayAmount() decimal.Decimal {
	return decimal.New(p.RealPrice, -2) // 单位为分，需要转为元
}
