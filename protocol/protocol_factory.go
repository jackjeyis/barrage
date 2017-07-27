package protocol

import "sync"

type ProtocolFactory struct {
	protocol_ map[string]Protocol
}

var (
	pf   *ProtocolFactory
	once sync.Once
)

func Instance() *ProtocolFactory {
	once.Do(func() {
		pf = &ProtocolFactory{
			protocol_: make(map[string]Protocol),
		}
		pf.Register("mqtt", new(MqttProtocol))
		pf.Register("barrage", NewBarrageProtocol())
	})
	return pf
}

func (pf *ProtocolFactory) Create(name string) Protocol {
	if proto, ok := pf.protocol_[name]; ok {
		return proto
	}
	return nil
}

func (pf *ProtocolFactory) Register(name string, proto Protocol) {
	pf.protocol_[name] = proto
}
