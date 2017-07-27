package timer

import (
	"barrage/util"
	"encoding/binary"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

type Timer struct {
	last_process_time int64
	per_tick          uint64
	total_ticks       uint64
	wheel             *TimingWheel
	ctrie             *util.Ctrie
	timer_id          uint64
}

var (
	pool sync.Pool
)

func (t *Timer) Incrment() uint64 {
	return atomic.AddUint64(&t.timer_id, 1)
}

func NewTimer(milli_per_tick uint64, ticks uint64) *Timer {
	timer := &Timer{
		per_tick:          milli_per_tick,
		total_ticks:       ticks,
		last_process_time: time.Now().UnixNano() / 1000000,
		ctrie:             util.New(nil),
	}
	pool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 8)
		},
	}
	timer.BuildWheel(ticks)
	return timer
}

func ToBytes(id uint64) []byte {
	buf := pool.Get().([]byte)
	binary.LittleEndian.PutUint64(buf, id)
	pool.Put(buf)
	return buf
}

func (t *Timer) AddTimer(milliseconds int64, data interface{}) uint64 {
	tid := t.Incrment()
	node := &TimingNode{data: data}
	t.ctrie.Insert(ToBytes(tid), node)
	node.ticks = (uint64)(time.Now().UnixNano()/1000000-t.last_process_time+milliseconds) / t.per_tick
	if node.ticks > t.total_ticks {
		node.ticks = 1
	}

	node.active = true
	node.next = nil
	t.wheel.AddNode(node.ticks, node)
	return tid
}

func (t *Timer) CancelTimer(timer_id uint64) error {
	node, ok := t.ctrie.Lookup(ToBytes(timer_id))
	if ok {
		n := node.(*TimingNode)
		n.active = false
		return nil
	}
	return errors.New("CancelTimer Error")
}

func (t *Timer) ProcessTimer() *TimingNode {
	current_process_time := time.Now().UnixNano() / 1000000
	ticks := (uint64)(current_process_time-t.last_process_time) / t.per_tick
	if ticks == 0 {
		return nil
	}

	if ticks > t.total_ticks {
		ticks = 1
	}

	t.last_process_time = current_process_time
	return t.wheel.Advance(ticks)
}

func (t *Timer) BuildWheel(total_ticks uint64) {
	t.wheel = NewTimingWheel(256, nil)
	total_ticks >>= 8
	current := t.wheel
	for total_ticks != 0 {
		next := NewTimingWheel(64, nil)
		total_ticks >>= 6
		current.BindNext(next)
		current = next
	}
}
