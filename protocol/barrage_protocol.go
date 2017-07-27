package protocol

import (
	"barrage/logger"
	"barrage/msg"
	"barrage/queue"
	"barrage/util/binary"
	"errors"
	"sync"
)

const (
	MaxBodySize = int32(1 << 20)
)

const (
	// size
	PackSize      = 4
	HeaderSize    = 2
	VerSize       = 2
	OpSize        = 4
	SeqSize       = 4
	RawHeaderSize = PackSize + HeaderSize + VerSize + OpSize + SeqSize
	MaxPackSize   = MaxBodySize + int32(RawHeaderSize)

	// offset
	PackOffset   = 0
	HeaderOffset = PackOffset + PackSize
	VerOffset    = HeaderOffset + HeaderSize
	OpOffset     = VerOffset + VerSize
	SeqOffset    = OpOffset + OpSize
)

var (
	ErrPackLen   = errors.New("server codec pack length error!")
	ErrHeaderLen = errors.New("server codec header length error!")
	pool         *sync.Pool
)

type Notify struct {
	Id   string `json:"id"`
	Ct   int    `json:"ct"`
	Rid  string `json:"roomId"`
	Uid  string `json:"fromId"`
	Code int8   `json:"code"`
	Time int64  `json:"time"`
}

type BarrageHeader struct {
	Ver int16
	Op  int32
	Seq int32
}

func (br *BarrageHeader) Decode(b *queue.IOBuffer) (int32, error) {
	temp := b.Read(uint64(RawHeaderSize))
	packLen := binary.BigEndian.Int32(temp[PackOffset:HeaderOffset])
	headerLen := binary.BigEndian.Int16(temp[HeaderOffset:VerOffset])
	br.Ver = binary.BigEndian.Int16(temp[VerOffset:OpOffset])
	br.Op = binary.BigEndian.Int32(temp[OpOffset:SeqOffset])
	br.Seq = binary.BigEndian.Int32(temp[SeqOffset:])
	if packLen > MaxPackSize {
		return 0, ErrPackLen
	}

	if headerLen != RawHeaderSize {
		return 0, ErrHeaderLen
	}
	bodyLen := packLen - int32(headerLen)
	return bodyLen, nil
}

func (br *BarrageHeader) Encode(b *queue.IOBuffer, rLen int32) error {
	/*buf := queue.Get()
	packLen := RawHeaderSize + rLen
	binary.BigEndian.PutInt32(buf, packLen)
	binary.BigEndian.PutInt16(buf, int16(RawHeaderSize))
	binary.BigEndian.PutInt16(buf, br.Ver)
	binary.BigEndian.PutInt32(buf, br.Op)
	binary.BigEndian.PutInt32(buf, br.Seq)
	b.Write(buf.Bytes())
	b.Write(buf[:])
	queue.Put(buf)
	*/
	packLen := RawHeaderSize + rLen
	buf := binary.Put(packLen,br.Op,br.Seq,br.Ver,RawHeaderSize)
	b.Write(buf[:])
	return nil
}

type Barrage struct {
	id      int
	channel msg.Channel
	BarrageHeader
	Body []byte
}

func (br *Barrage) Type() uint8 {
	return uint8(0)
}

func (br *Barrage) SetChannel(ch msg.Channel) {
	br.channel = ch
}

func (br *Barrage) Channel() msg.Channel {
	return br.channel
}

func (br *Barrage) SetHandlerId(i int) {
	br.id = i
}

func (br *Barrage) HandlerId() int {
	return br.id
}
func (br *Barrage) Decode(b *queue.IOBuffer, rLen int32) (err error) {
	br.Body = b.Read(uint64(rLen))
	return nil
}

func (br *Barrage) Encode(b *queue.IOBuffer) error {
	err := br.BarrageHeader.Encode(b, int32(len(br.Body)))
	if err != nil {
		return err
	}
	if br.Body != nil {
		b.Write(br.Body)
	}
	return nil
}

type BarrageProtocol struct {
	BarrageHeader
	rLen int32
}

func NewBarrageProtocol() *BarrageProtocol {
	pool = &sync.Pool{
		New: func() interface{} {
			return &Barrage{}
		},
	}
	return &BarrageProtocol{}
}

func (b *BarrageProtocol) Encode(msg msg.Message, buf *queue.IOBuffer) error {
	//pool.Put(msg)
	return msg.Encode(buf)
}

func (b *BarrageProtocol) Decode(buf *queue.IOBuffer) (msg.Message, error) {
	var (
		rLen int32
		err  error
	)
	if buf.Init() {
		if buf.GetReadSize() < uint64(RawHeaderSize) {
			return nil, HeaderErr
		}
		rLen, err = b.BarrageHeader.Decode(buf)
		buf.SetInit(false)
	} else {
		rLen = b.rLen
	}

	if uint64(rLen) > buf.GetReadSize() {
		b.rLen = rLen
		return nil, BodyErr
	}

	if err != nil {
		logger.Error("BarrageProtocol.BarrageHeader.Decode error %v", err)
		return nil, err
	}
	return decodeBarrageMessage(b.BarrageHeader, buf, rLen)
}

func decodeBarrageMessage(header BarrageHeader, b *queue.IOBuffer, rLen int32) (msg.Message, error) {

	//barrage, _ := pool.Get().(*Barrage)
	//barrage.BarrageHeader = header
	barrage := &Barrage{BarrageHeader: header}
	err := barrage.Decode(b, rLen)
	return barrage, err
}
