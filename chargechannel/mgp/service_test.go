package mgp_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/babybabylong/first-business/chargechannel/mgp"
	"github.com/fighterlyt/log"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

var (
	service *mgp.Service
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
	service = mgp.NewService(`https://mgp-pay.com:8443`, `c0b13dbb814e4a5297f97e6f5ee0aabf`, `API21337616880378620`, &http.Client{}, logger, time.Second, "0")
}

func TestService_CreateOrderNo(t *testing.T) {
	TestService(t)

	orderNo := service.CreateOrderNo(0, decimal.New(10, 0))

	require.NoError(t, service.CreateOrder(context.Background(), orderNo, decimal.New(10, 0), `http://baidu.com`))
}
