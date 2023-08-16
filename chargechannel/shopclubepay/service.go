package shopclubepay

// shopClub ePay支付

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/babybabylong/common/helpers"
	"github.com/babybabylong/first-business/chargechannel"
	"github.com/fighterlyt/log"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

// Service 服务
type Service struct {
	host       string // 服务地址，包括schema 地址 端口
	privateKey string // 商户私钥
	mchID      int    // 商户id
	appID      int
	client     *http.Client             // http 客户端
	logger     log.Logger               // 日志器
	timeout    time.Duration            // http请求超时
	channelKey chargechannel.ChannelKey // 充值渠道key
}

func NewService(host, privateKey string, mchID, appID int, client *http.Client, logger log.Logger, timeout time.Duration, channelKey chargechannel.ChannelKey) *Service {
	return &Service{
		host:       host,
		privateKey: privateKey,
		mchID:      mchID,
		appID:      appID,
		client:     client,
		logger:     logger,
		timeout:    timeout,
		channelKey: channelKey,
	}
}

func (s Service) Key() chargechannel.ChannelKey {
	return s.channelKey
}

func (s Service) PrivateKey() string {
	return s.privateKey
}

func (s Service) NeedCheck() (template chargechannel.AsyncCallBackTemplate, need bool) {
	return &payAsyncResponse{}, false
}

func (s Service) CreateOrderNo(id int64, amount decimal.Decimal) string {
	return primitive.NewObjectID().Hex()
}

func (s Service) CreateOrder(ctx context.Context, orderNo string, amount decimal.Decimal, callbackURL string, extend *chargechannel.CreateOrderExtendParam) (payUrl, payHtml string, err error) {
	if extend == nil {
		return "", "", errors.New("参数不足")
	}

	if payUrl, err = s.charge(ctx, orderNo, extend.PayCode, amount, extend.UserIP, extend.UserID, extend.SuccessURL, callbackURL); err != nil {
		return "", "", errors.Wrap(err, "下单失败")
	}

	return payUrl, "", nil
}

func (s Service) charge(ctx context.Context, orderNo string, payCode int, amount decimal.Decimal, userIP string, userID int64, successURL, callbackURL string) (payURL string, err error) {
	// 1. 校验参数
	if amount.LessThanOrEqual(decimal.Zero) {
		return payURL, errors.New(`订单金额必须大于0`)
	}

	if !helpers.IsURL(callbackURL) || !helpers.IsURL(successURL) {
		return payURL, fmt.Errorf(`非法的回调地址[%s]`, callbackURL)
	}

	// 2. 准备请求相关
	var (
		logger     = helpers.GetLogger(ctx, s.logger)
		cancel     context.CancelFunc
		body       io.Reader
		resp       *http.Response
		result     = &payResponse{}
		resultCode = &payResponseCode{}
	)

	ctx, cancel = context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// 3. 准备请求参数
	if body, err = s.prepareArgument(orderNo, payCode, amount, userIP, userID, successURL, callbackURL); err != nil {
		return payURL, errors.Wrap(err, `准备参数`)
	}

	// 5. 执行HTTP请求
	if resp, err = s.doHTTPRequest(ctx, body); err != nil {
		return payURL, errors.Wrap(err, `执行请求`)
	}

	// 6. 处理应答
	defer func() {
		_ = resp.Body.Close()
	}()

	var (
		response []byte
	)

	if response, err = ioutil.ReadAll(resp.Body); err != nil {
		return payURL, errors.Wrap(err, `读取应答`)
	}

	logger.Info(`读取到应答`, zap.ByteString(`应答`, response))

	if resp.StatusCode != http.StatusOK {
		return payURL, errors.New(resp.Status)
	}

	if err = json.Unmarshal(response, resultCode); err != nil {
		return payURL, errors.Wrap(err, `解析应答错误`)
	}

	logger.Info(`应答解析成功`, zap.Any(`应答`, resultCode))

	if err = resultCode.Validate(); err != nil {
		return payURL, err
	}

	if err = json.Unmarshal(response, result); err != nil {
		return payURL, errors.Wrap(err, `解析应答错误`)
	}

	logger.Info(`应答解析成功`, zap.Any(`应答`, result))

	return result.Data.PayURL, nil
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

	if req, err = http.NewRequestWithContext(ctx, http.MethodPost, s.host+`/payApi/PayApi/CreateOrder`, body); err != nil {
		return nil, errors.Wrap(err, `构建请求`)
	}

	req.Header.Set(`Content-Type`, `application/json`)

	if resp, err = s.client.Do(req); err != nil {
		return nil, errors.Wrap(err, `执行请求`)
	}

	return resp, nil
}

func (s Service) prepareArgument(orderNo string, payCode int, amount decimal.Decimal, userIP string, userID int64, successURL, callbackURL string) (reader io.Reader, err error) { //nolint:lll
	price := amount.Mul(decimal.NewFromInt(100)).IntPart() // 单位为分
	argument := newPayArgument(s.mchID, payCode, orderNo, price, s.appID, userIP, fmt.Sprintf("%d", userID), callbackURL, successURL)

	argument.Sign = argument.sign(s.privateKey)

	var (
		body []byte
	)

	if body, err = json.Marshal(argument); err != nil {
		return nil, errors.Wrap(err, `json序列化`)
	}

	s.logger.Info("请求参数", zap.String("body", string(body)))

	return bytes.NewReader(body), nil
}

func (s Service) Check(channelOrderNo string) (paid chargechannel.PaidStatus, err error) {
	return chargechannel.PaidUnknown, chargechannel.ErrNotSupported
}
