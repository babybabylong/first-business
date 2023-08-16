package shopclubepay

import (
	"context"
	"github.com/babybabylong/first-business/chargechannel"
	"github.com/fighterlyt/log"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"testing"
	"time"
)

var (
	service *Service
	err     error
	logger  log.Logger
)

func TestMain(m *testing.M) {
	if logger, err = log.NewEasyLogger(true, false, ``, `test`); err != nil {
		panic(err.Error())
	}

	os.Exit(m.Run())
}

func TestService(t *testing.T) {
	service = NewService("https://asdqw3ds8e3wj80opd-order.xnslxxl.com", "5516ec2da46b080c26ca04b0faee6537", 152, 52, &http.Client{}, logger, time.Second*5, chargechannel.ChannelKeyEPayRuble)
}

func TestService_CreateOrderNo(t *testing.T) {
	TestService(t)

	orderNo := service.CreateOrderNo(0, decimal.New(10, 0))

	payUrl, _, err := service.CreateOrder(context.Background(), orderNo, decimal.NewFromInt(1001), `http://baidu.com`, &chargechannel.CreateOrderExtendParam{
		PayCode:    70104,
		UserID:     1,
		UserIP:     "47.91.120.169",
		SuccessURL: "http://baidu.com",
	})

	require.NoError(t, err)

	t.Log("payURl:", payUrl)
}
