package kab

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/babybabylong/common/helpers"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"io/ioutil"
	"net/url"
	"strings"
	"time"

	"github.com/babybabylong/first-business/chargechannel"
	"github.com/fighterlyt/log"
	"github.com/shopspring/decimal"
	"net/http"
)

type Service struct {
	host        string        // 服务地址，包括schema 地址 端口
	apiKey      string        // 商户私钥
	client      *http.Client  // http 客户端
	logger      log.Logger    // 日志器
	timeout     time.Duration // http请求超时
	channelCode string        // 通道代码
	payMethod   string        // 支付方式
}

func NewService(host, apiKey, channelCode, payMethod string, timeout time.Duration, logger log.Logger) *Service {
	return &Service{
		host:        host,
		apiKey:      apiKey,
		client:      &http.Client{},
		logger:      logger,
		timeout:     timeout,
		channelCode: channelCode,
		payMethod:   payMethod,
	}
}

func (s Service) Key() chargechannel.ChannelKey {
	return chargechannel.ChannelKeyKab
}

func (s Service) PrivateKey() string {
	return s.apiKey
}

func (s Service) NeedCheck() (template chargechannel.AsyncCallBackTemplate, need bool) {
	return &payAsyncResponse{}, true
}

func (s Service) CreateOrderNo(_ int64, amount decimal.Decimal) string {
	return s.generateChannelOrderNo(amount)
}

func (s Service) generateChannelOrderNo(_ decimal.Decimal) string {
	return primitive.NewObjectID().Hex()
}

func (s Service) CreateOrder(ctx context.Context, orderNo string, amount decimal.Decimal, callbackURL string, extend *chargechannel.CreateOrderExtendParam) (payUrl, payHtml string, err error) {
	if payUrl, err = s.charge(ctx, orderNo, amount, callbackURL); err != nil {
		return "", "", errors.Wrap(err, "下单失败")
	}

	return payUrl, "", nil
}

func (s Service) charge(ctx context.Context, orderNo string, amount decimal.Decimal, callbackURL string) (payUrl string, err error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return payUrl, errors.New(`订单金额必须大于0`)
	}

	if !helpers.IsURL(callbackURL) {
		return payUrl, fmt.Errorf(`非法的回调地址[%s]`, callbackURL)
	}

	// 2. 准备请求相关
	var (
		logger       = helpers.GetLogger(ctx, s.logger)
		argument     url.Values
		req          *http.Request
		resp         *http.Response
		result       = &chargeResponse{}
		responseByte []byte
	)

	if argument, err = s.prepareArgument(orderNo, amount, callbackURL, logger); err != nil {
		return "", errors.Wrap(err, "构造请求")
	}

	logger.Info(`获取支付参数`, zap.String(`参数`, argument.Encode()))

	if req, err = http.NewRequestWithContext(ctx, http.MethodGet, s.host+`/pay_index.php?`+argument.Encode(), nil); err != nil {
		return "", errors.Wrap(err, `构建请求`)
	}

	if resp, err = s.client.Do(req); err != nil {
		return "", errors.Wrap(err, `执行请求`)
	}

	if responseByte, err = ioutil.ReadAll(resp.Body); err != nil {
		return "", errors.Wrap(err, `读取应答`)
	}

	logger.Info(`读取到应答`, zap.ByteString(`应答`, responseByte))

	if resp.StatusCode != http.StatusOK {
		return "", errors.New(resp.Status)
	}

	if err = json.Unmarshal(responseByte, result); err != nil {
		return "", errors.Wrap(err, `解析应答错误`)
	}

	logger.Info(`应答解析成功`, zap.Any(`应答`, result))

	if err = result.Validate(); err != nil {
		return payUrl, errors.Wrap(err, "下单失败")
	}

	return result.Data.PayURL, nil
}

func (s Service) prepareArgument(orderNo string, amount decimal.Decimal, callbackURL string, logger log.Logger) (url.Values, error) {
	var argument = url.Values{}

	argument.Set("u", s.channelCode)
	argument.Set("id", orderNo)
	argument.Set("je", amount.Mul(decimal.New(100, 0)).String()) // 以分为单位
	argument.Set("sp", url.QueryEscape(fmt.Sprintf("充值%sU", amount)))
	argument.Set("cb", callbackURL)
	argument.Set("json", "1")

	// 签名：md5(u+id+je+sp+apikey)
	signStr := strings.Join([]string{argument.Get("u"), argument.Get("id"), argument.Get("je"), argument.Get("sp"), s.apiKey}, "")
	argument.Set("sign", sign(signStr))

	return argument, nil
}

func (s Service) Check(channelOrderNo string) (paid chargechannel.PaidStatus, err error) {
	return chargechannel.PaidUnknown, chargechannel.ErrNotSupported
}
