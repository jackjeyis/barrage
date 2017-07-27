package msg

import (
	"barrage/queue"
)

type Message interface {
	IOMessage
	Type() uint8
	SetChannel(Channel)
	Channel() Channel
	HandlerId() int
	SetHandlerId(int)
}

type IOMessage interface {
	Encode(b *queue.IOBuffer) error
	Decode(b *queue.IOBuffer, len int32) error
}
