package kab

import (
	"context"
	"github.com/fighterlyt/log"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"os"
	"testing"
	"time"
)

var (
	service *Service
	logger  log.Logger
	err     error
)

func TestMain(m *testing.M) {
	if logger, err = log.NewEasyLogger(true, false, ``, `test`); err != nil {
		panic(err.Error())
	}

	service = NewService(`https://four2.kabproducts.online`, `5QzzwZh4MjbexiOiW1Guz19Fm6Xe0JOq`, `97`, "c3301", time.Second*10, logger)

	os.Exit(m.Run())
}

func TestService_CreateOrder(t *testing.T) {
	orderNo := primitive.NewObjectID().Hex()

	var payUrl string
	payUrl, _, err = service.CreateOrder(context.Background(), orderNo, decimal.New(10, 0), `http://baidu.com`, nil)

	require.NoError(t, err)

	t.Log(payUrl)
}
