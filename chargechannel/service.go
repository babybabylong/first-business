package chargechannel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/babybabylong/common/helpers"
	"github.com/fighterlyt/log"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type Service struct {
	manager  Manager
	logger   log.Logger
	engine   *gin.Engine
	accessor Accessor
	baseURL  string // http基础路径，baseURL+/1/1 就可以调用到httpOnCallBack
}

func NewService(manager Manager, logger log.Logger, engine *gin.Engine, accessor Accessor, baseURL string) *Service {
	return &Service{manager: manager, logger: logger, engine: engine, accessor: accessor, baseURL: baseURL}
}

// StartEPayCallback shop-club的http链接
func (s Service) StartEPayCallback(prefix string) {
	s.engine.POST(fmt.Sprintf(`/%s/:key/:orderNo`, prefix), s.httpOnCallBack)
}

func (s Service) Start() {
	s.engine.POST(`/:key/:orderNo`, s.httpOnCallBack)
	s.engine.GET(`/callback/:key/:orderNo`, s.httpOnCallBack) // kab渠道的回调
}

func (s Service) httpOnCallBack(ctx *gin.Context) {
	channelKeyStr := ctx.Param(`key`)
	orderNo := ctx.Param(`orderNo`)

	channelKey, err := strconv.Atoi(channelKeyStr)
	if err != nil {
		ctx.String(http.StatusOK, err.Error())
		return
	}

	var result io.Reader

	switch ChannelKey(channelKey) {
	case ChannelKeyEPay, ChannelKeyEPayRuble, ChannelKeyEPayU:
		result, err = s.OnCallBack(ChannelKey(channelKey), orderNo, ctx.Request.Body)
	case ChannelKeyKab:
		result, err = s.OnCallBackKab(ctx, ChannelKey(channelKey), orderNo)
	default:
		ctx.String(http.StatusOK, fmt.Sprintf("不支持的充值渠道%s", ChannelKey(channelKey).Text()))
		return
	}

	if err != nil {
		ctx.String(http.StatusOK, err.Error())
		return
	}

	var (
		resp []byte
	)

	if result != nil {
		resp, _ = io.ReadAll(result)
	}

	ctx.String(http.StatusOK, string(resp))
}

func (s Service) OnCallBackKab(ctx *gin.Context, channelKey ChannelKey, orderNo string) (result io.Reader, err error) {
	data := fmt.Sprintf(`{"orderid":"%s","amount":"%s","payno":"%s","sign":"%s"}`, ctx.Query("orderid"), ctx.Query("amount"), ctx.Query("payno"), ctx.Query("sign"))
	body := ioutil.NopCloser(strings.NewReader(data))

	return s.OnCallBack(channelKey, orderNo, body)
}

func (s Service) OnCallBack(channelKey ChannelKey, orderNo string, body io.ReadCloser) (result io.Reader, err error) {
	if body != nil {
		defer func() {
			_ = body.Close()
		}()
	}

	var (
		template AsyncCallBackTemplate
	)

	if template, err = s.manager.LoadTemplateBy(channelKey); err != nil {
		return nil, errors.Wrap(err, `加载模板`)
	}

	resp := template.New()

	if err = json.NewDecoder(body).Decode(resp); err != nil {
		return resp.Result(), errors.Wrap(err, `解码失败`)
	}

	channel, err := s.manager.LoadByKey(channelKey)
	if err != nil {
		return nil, errors.Wrap(err, "加载渠道失败")
	}

	if err = resp.Validate(channel.PrivateKey()); err != nil {
		return resp.Result(), errors.Wrap(err, `验证失败`)
	}

	switch resp.Status() {
	case Paid:
		if setErr := s.accessor.SetRecordFinish(channelKey, orderNo, resp.RealPayAmount(), nil); setErr != nil {
			s.logger.Error(`设置支付状态失败`, helpers.ZapError(setErr))
		}
	case PaidFail:
		if setErr := s.accessor.SetRecordFinish(channelKey, orderNo, decimal.Zero, errors.New(`回调通知支付失败`)); setErr != nil {
			s.logger.Error(`设置支付状态失败`, helpers.ZapError(setErr))
		}
	}

	return resp.Result(), nil
}

func (s Service) Charge(ctx context.Context, id int64, amount decimal.Decimal, channelKey ChannelKey, extend *CreateOrderExtendParam) (payUrl, payHtml string, err error) {
	channel, err := s.manager.LoadByKey(channelKey)
	if err != nil {
		return payUrl, payHtml, errors.Wrap(err, `加载渠道`)
	}

	channelOrderNo := channel.CreateOrderNo(id, amount)

	callbackURL := s.generateCallBackURL(channelKey, channelOrderNo)

	payUrl, payHtml, err = channel.CreateOrder(ctx, channelOrderNo, amount, callbackURL, extend)

	if setErr := s.accessor.SetRecordStarted(id, channelOrderNo, err); setErr != nil {
		s.logger.Error(`保存订单发起状态失败`, helpers.ZapError(setErr))
	}

	return payUrl, payHtml, err
}

func (s Service) generateCallBackURL(channelKey ChannelKey, orderNo string) string {
	if channelKey == ChannelKeyKab {
		return s.baseURL + `/callback/` + fmt.Sprintf("%d", channelKey) + `/` + orderNo
	}

	return s.baseURL + `/` + fmt.Sprintf("%d", channelKey) + `/` + orderNo
}
