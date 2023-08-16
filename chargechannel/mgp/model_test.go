package mgp

import (
	"testing"

	"github.com/shopspring/decimal"
)

func Test_payArgument_sign(t *testing.T) {
	type args struct {
		privateKey string
	}

	tests := []struct {
		name   string
		fields payArgument
		args   args
		want   string
	}{
		{
			name: `文档示例测试`,
			fields: payArgument{
				Version:     version,
				SignType:    signType,
				MerchantNo:  `API2442810283706600`,
				Date:        "20191127172151",
				ChannelType: `0`,
				NoticeURL:   `https://www.baidu.com/`,
				OrderNo:     `1574846511910`,
				Amount:      decimal.New(10, 0),
			},
			args: args{privateKey: `2cf6782df6d7478f872954ef8ff45a16`},
			want: `c9937998ef046127f762a002a6e8be8a`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fields.sign(tt.args.privateKey); got != tt.want {
				t.Errorf("sign() = %v, want %v", got, tt.want)
			}
		})
	}
}
