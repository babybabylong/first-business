package chargechannel

import (
	"github.com/babybabylong/common/model"
	"github.com/youthlin/t"
)

type ChannelKey int

func (c ChannelKey) Value() int {
	return int(c)
}

var (
	_ = t.T("全部")
	_ = t.T("ERC20")
	_ = t.T("TRC20")
	_ = t.T("E-Pay")
	_ = t.T("线下银行卡转账")
	_ = t.T("kab法币转账")
	_ = t.T("银商渠道")
)

func (c ChannelKey) Text() string {
	switch c {
	case ChannelKeyAll:
		return "全部"
	case ChannelKeyErc:
		return "ERC20"
	case ChannelKeyTrc:
		return "TRC20"
	case ChannelKeyEPay:
		return "E-Pay"
	case ChannelKeyBank:
		return "银行卡支付"
	case ChannelKeyKab:
		return "kab法币转账"
	case ChannelKeyMerchant:
		return "银商渠道"
	case ChannelKeyEPayRuble:
		return "Eapy(卢布)"
	case ChannelKeyEPayU:
		return "Eapy(U)"
	default:
		return "未知"
	}
}

func (c ChannelKey) MarshalJSON() ([]byte, error) {
	return model.MarshalJSON(c)
}

const (
	ChannelKeyAll       ChannelKey = 0
	ChannelKeyErc       ChannelKey = 1  // erc
	ChannelKeyTrc       ChannelKey = 2  // trc
	ChannelKeyEPay      ChannelKey = 3  // E-Pay(地平线)
	ChannelKeyBank      ChannelKey = 4  // 线下银行卡转账(地平线)
	ChannelKeyKab       ChannelKey = 5  // kab法币转账(zalaro)
	ChannelKeyMerchant  ChannelKey = 6  // shop_club银商渠道
	ChannelKeyEPayRuble ChannelKey = 7  // shop_club Eapy(卢布)
	ChannelKeyEPayU     ChannelKey = 8  // shop_club Eapy(U)
	ChannelKeyProxyRUR  ChannelKey = 9  // shop_club proxypay RUR
	ChannelKeyProxyUSDT ChannelKey = 10 // shop_club proxypay USDT
)

func (c ChannelKey) Protocol() model.Protocol {
	protocol := model.Trc20
	if c == ChannelKeyErc {
		protocol = model.Erc20
	}

	return protocol
}
