package chargechannel

import (
	"errors"
	"fmt"
	"sync"
)

const (
	initCapacity = 10
)

// manager 充值渠道管理器
type manager struct {
	channels  map[ChannelKey]Channel
	lock      *sync.RWMutex
	templates map[ChannelKey]AsyncCallBackTemplate
}

/*NewManager 新建充值渠道管理器
参数:
返回值:
*	Manager	Manager	管理器
*/
func NewManager() Manager {
	return &manager{
		lock:      &sync.RWMutex{},
		channels:  make(map[ChannelKey]Channel, initCapacity),
		templates: make(map[ChannelKey]AsyncCallBackTemplate, initCapacity),
	}
}

func (m manager) Register(channel Channel) error {
	if channel == nil {
		return errors.New(`充值渠道不能为空`)
	}

	key := channel.Key()

	if key == 0 {
		return errors.New(`key不能为空`)
	}

	m.lock.Lock()

	defer m.lock.Unlock()

	if _, exist := m.channels[key]; exist {
		return fmt.Errorf(`key[%s]重复`, key)
	}

	m.channels[key] = channel

	template, _ := channel.NeedCheck()

	if template != nil {
		m.templates[key] = template
	}

	return nil
}

func (m manager) LoadByKey(key ChannelKey) (channel Channel, err error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if data, exist := m.channels[key]; exist {
		return data, nil
	}

	return nil, fmt.Errorf(`key[%s]的渠道不存在`, key)
}

func (m manager) LoadTemplateBy(key ChannelKey) (template AsyncCallBackTemplate, err error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if data, exist := m.templates[key]; exist {
		return data, nil
	}

	return nil, fmt.Errorf(`key[%d]的渠道不存在`, key)
}
