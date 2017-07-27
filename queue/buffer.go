package queue

import (
	"errors"
	"strconv"
	"sync/atomic"
)

type IOBuffer struct {
	buf  []byte
	wt   uint64
	rt   uint64
	size uint64
	init bool
}

func NewIOBuffer(it bool) *IOBuffer {
	return &IOBuffer{init: it}
}

func (b *IOBuffer) EnsureWrite(size uint64) (uint64, error) {
	data_bytes := b.wt - b.rt
	if b.rt >= size {
		if data_bytes == 0 {
			b.wt = 0
			b.rt = 0
			return b.wt, nil
		} else if b.rt > 1024*1024 {
			copy(b.buf, b.buf[b.rt:b.wt])
			b.wt = data_bytes
			b.rt = 0
			return b.wt, nil
		}
	}
	remain_bytes := b.size - b.wt
	if remain_bytes >= size {
		return b.wt, nil
	}

	for b.size < b.wt+size {
		if b.size == 0 {
			b.size = 1024
		} else {
			b.size *= 2
		}
	}

	if b.size > 0xFFFFFFFF {
		return 0, errors.New("Buffer Overflow")
	}

	buffer := make([]byte, b.size)
	if len(b.buf) != 0 {
		copy(buffer, b.buf[:])
	}
	b.buf = buffer
	return b.wt, nil
}

func (b *IOBuffer) Write(bs []byte) error {
	size := uint64(len(bs))
	if size == 0 {
		return nil
	}
	wstart, err := b.EnsureWrite(size)
	if err != nil {
		return err
	}
	copy(b.buf[wstart:wstart+size], bs)
	b.Produce(size)
	return nil
}

func (b *IOBuffer) Read(n uint64) []byte {
	var (
		buffer = make([]byte,n)
	)
	if n > b.GetReadSize() {
		copy(buffer,b.buf[b.rt:])
		b.Consume(b.GetReadSize())
	} else {
		copy(buffer,b.buf[b.rt : b.rt+n])
		b.Consume(n)
	}
	return buffer
}

func (b *IOBuffer) Produce(size uint64) {
	atomic.AddUint64(&b.wt,size)
}

func (b *IOBuffer) Consume(size uint64) {
	atomic.AddUint64(&b.rt,size)
}

func (b *IOBuffer) GetReadSize() uint64 {
	return b.wt - b.rt
}

func (b *IOBuffer) GetRead() uint64 {
	return b.rt
}

func (b *IOBuffer) GetWrite() uint64 {
	return b.wt
}

func (b *IOBuffer) Buffer() []byte {
	return b.buf
}

func (b *IOBuffer) Len() uint64 {
	return uint64(len(b.buf))
}

func (b *IOBuffer) Byte(cnt uint64) byte {
	return b.buf[b.rt+cnt]
}

func (b *IOBuffer) Reset() {
	b.wt = 0
	b.rt = 0
	b.buf = b.buf[:0]
}

func (b *IOBuffer) Init() bool {
	return b.init
}

func (b *IOBuffer) SetInit(init bool) {
	b.init = init
}

func (b *IOBuffer) Summary() string {
	return "[Size:" + strconv.FormatUint(b.size, 10) + ",Wt:" + strconv.FormatUint(b.wt, 10) + ",Rt:" + strconv.FormatUint(b.rt, 10) + "]"
}

/*func main() {
	buf := NewIOBuffer()
	buf.Write([]byte(strings.Repeat("hello", 1024)))
	//fmt.Println(string(buf.buf[:buf.GetReadSize()]))
	fmt.Println(buf.size, buf.wt)
}*/
