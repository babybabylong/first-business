package mgp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/babybabylong/common/helpers"
	"github.com/babybabylong/first-business/chargechannel"
	"github.com/fighterlyt/log"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// Service 服务
type Service struct {
	host        string        // 服务地址，包括schema 地址 端口
	privateKey  string        // 商户私钥
	merchantNo  string        // 商户号
	client      *http.Client  // http 客户端
	logger      log.Logger    // 日志器
	timeout     time.Duration // http请求超时
	ChannelType ChannelType   // 通道类型
}

func NewService(host, privateKey, merchantNo string, client *http.Client, logger log.Logger, timeout time.Duration, channelType ChannelType) *Service {
	return &Service{
		host:        host,
		privateKey:  privateKey,
		merchantNo:  merchantNo,
		client:      client,
		logger:      logger,
		timeout:     timeout,
		ChannelType: channelType,
	}
}

func (s Service) PrivateKey() string {
	return s.privateKey
}

/*Key 支付渠道的key
参数:
返回值:
*	string	string	key
*/
func (s Service) Key() chargechannel.ChannelKey {
	return chargechannel.ChannelKeyEPay
}

/*NeedCheck 是否需要主动查单
参数:
返回值:
*	template	chargechannel.AsyncCallBackTemplate	异步查单模板
*	need    	bool                               	是否需要主动查单
*/
func (s Service) NeedCheck() (template chargechannel.AsyncCallBackTemplate, need bool) {
	return &payAsyncResponse{}, false
}

/*CreateOrder 创建订单
参数:
*	ctx           	context.Context	上限文
*	_             	int64          	订单ID(无用)
*	amount        	decimal.Decimal	订单金额
*	callbackURL   	string         	回调地址
返回值:
*	channelOrderNo	string         	商户订单号(生成)
*	err           	error          	错误
*/
func (s Service) CreateOrder(ctx context.Context, orderNo string, amount decimal.Decimal, callbackURL string, extend *chargechannel.CreateOrderExtendParam) (payUrl, payHtml string, err error) { //nolint:lll
	return s.charge(ctx, orderNo, amount, callbackURL)
}

func (s Service) CreateOrderNo(_ int64, amount decimal.Decimal) string {
	return s.generateChannelOrderNo(amount)
}

/*Check 主动查询支付状态，本实现不需要
参数:
*	_   	string                  	参数1
返回值:
*	paid	chargechannel.PaidStatus	返回值1
*	err 	error                   	返回值2
*/
func (s Service) Check(_ string) (paid chargechannel.PaidStatus, err error) {
	return chargechannel.PaidUnknown, chargechannel.ErrNotSupported
}

/*charge 获取充值订单
参数:
*	ctx           	context.Context	上下文
*	amount        	decimal.Decimal	金额
*	callbackURL   	string         	回调地址
返回值:
*	channelOrderNo	string         	商户订单号(实现生成,并非服务端返回)
*	err           	error           错误
*/
func (s Service) charge(ctx context.Context, orderNo string, amount decimal.Decimal, callbackURL string) (payUrl, payHtml string, err error) { //nolint:lll
	// 1. 校验参数
	if amount.LessThanOrEqual(decimal.Zero) {
		return payUrl, payHtml, errors.New(`订单金额必须大于0`)
	}

	if !helpers.IsURL(callbackURL) {
		return payUrl, payHtml, fmt.Errorf(`非法的回调地址[%s]`, callbackURL)
	}

	var (
		body   io.Reader
		resp   *http.Response
		result = &payResponse{}
		cancel context.CancelFunc
	)

	// 2. 准备请求相关
	logger := helpers.GetLogger(ctx, s.logger)

	ctx, cancel = context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// 3. 准备请求参数
	if body, err = s.prepareArgument(amount, callbackURL, orderNo); err != nil {
		return payUrl, payHtml, errors.Wrap(err, `准备参数`)
	}

	// 5. 执行HTTP请求
	if resp, err = s.doHTTPRequest(ctx, body); err != nil {
		return payUrl, payHtml, errors.Wrap(err, `执行请求`)
	}

	// 6. 处理应答
	if err = s.processResult(logger, resp, result); err != nil {
		return payUrl, payHtml, errors.Wrap(err, `解析应答错误`)
	}

	if err = result.Validate(); err != nil {
		return payUrl, payHtml, nil
	}

	// 如果code=0 先判断PayURL，如果为空字符串，则取PayHtml
	if result.Detail.PayURL != "" {
		return result.Detail.PayURL, "", nil
	}

	return "", result.Detail.PayHTML, nil
}

func (s Service) processResult(logger log.Logger, resp *http.Response, value interface{}) error {
	defer func() {
		_ = resp.Body.Close()
	}()

	var (
		response []byte
		err      error
	)

	if response, err = ioutil.ReadAll(resp.Body); err != nil {
		return errors.Wrap(err, `读取应答`)
	}

	logger.Info(`读取到应答`, zap.ByteString(`应答`, response))

	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}

	if err = json.Unmarshal(response, value); err != nil {
		return errors.Wrap(err, `解析应答错误`)
	}

	logger.Info(`应答解析成功`, zap.Any(`应答`, value))

	return nil
}

/*doHTTPRequest 执行http请求
参数:
*	ctx 	context.Context	上下文
*	body	io.Reader      	body
返回值:
*	resp	*http.Response 	应答
*	err 	error          	错误
*/
func (s Service) doHTTPRequest(ctx context.Context, body io.Reader) (resp *http.Response, err error) {
	var (
		req *http.Request
	)

	if req, err = http.NewRequestWithContext(ctx, http.MethodPost, s.host+`/api/pay/V2`, body); err != nil {
		return nil, errors.Wrap(err, `构建请求`)
	}

	req.Header.Set(`Content-Type`, `application/json`)

	if resp, err = s.client.Do(req); err != nil {
		return nil, errors.Wrap(err, `执行请求`)
	}

	return resp, nil
}

func (s Service) prepareArgument(amount decimal.Decimal, callbackURL, channelOrderNo string) (reader io.Reader, err error) { //nolint:lll
	argument := newPayArgument(s.merchantNo, callbackURL, channelOrderNo, amount, s.ChannelType)

	argument.Sign = argument.sign(s.privateKey)

	var (
		body []byte
	)

	if body, err = json.Marshal(argument); err != nil {
		return nil, errors.Wrap(err, `json序列化`)
	}

	return bytes.NewReader(body), nil
}

func (s Service) generateChannelOrderNo(amount decimal.Decimal) string {
	amountStr := amount.StringFixed(amountPrecision)

	if len(amountStr) > orderNoMaxLength-objectIDLength {
		amountStr = amountStr[:orderNoMaxLength-objectIDLength]
	}

	return primitive.NewObjectID().Hex() + amountStr
}
