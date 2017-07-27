package protocol

import (
	"barrage/msg"
	"barrage/queue"
)

type Protocol interface {
	Encode(msg msg.Message, buf *queue.IOBuffer) error
	Decode(buf *queue.IOBuffer) (msg.Message, error)
}
