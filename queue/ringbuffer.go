package queue

import "errors"

type Item struct {
	id   uint64
	Data interface{}
}

type RingBuffer struct {
	size uint64
	mask uint64
	data []*Item
}

func NewRingBuffer(size uint64) (*RingBuffer, error) {
	if size&(size-1) != 0 {
		return nil, errors.New("error buffer size provide!")
	}
	ring := &RingBuffer{
		size: size,
		mask: size - 1,
		data: make([]*Item, size),
	}
	var i uint64
	for i = 0; i < size; i++ {
		ring.data[i] = &Item{}
	}
	return ring, nil
}

func (r *RingBuffer) At(index uint64) *Item {
	return r.data[index&r.mask]
}

func (r *RingBuffer) Size() uint64 {
	return r.size
}

func (r *RingBuffer) Index(index uint64) uint64 {
	return index & r.mask
}

func (r *RingBuffer) GetDistance(size, begin, end uint64) uint64 {
	begin = r.Index(begin)
	end = r.Index(end)
	if end >= begin {
		return end - begin
	} else {
		return size - (begin - end)
	}
}
