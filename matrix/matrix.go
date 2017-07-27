package matrix

import (
	"barrage/logger"
	"barrage/queue"
	"barrage/util"
	"errors"
	"strconv"
	"sync/atomic"
	"time"
)

type Mode uint8

const (
	kModeLazy = Mode(iota)
	kModeAutoSuccess
	kModeAtuoFail
)

const (
	kSet uint8 = iota
	kAdd
	kSub
	kReset
	kTimeDistribute
)

var collector *MatrixCollector

func Init(bucket uint32, size uint64) error {
	if size&(size-1) != 0 {
		return errors.New("size not 2 pow!")
	}
	collector = NewMatrixCollector(bucket, size)
	/*if err := collector.Start(); err != nil {
		logger.Info("collector.Start Error %v", err)
		return err
	}*/
	return nil
}

func Stop() {
	atomic.StoreUint64(&collector.running, 0)
}

func SendToCollector(item *MatrixItem) error {
	if err := collector.Send(item); err != nil {
		logger.Warn("SendToCollector Error %v", err)
		return err
	}
	return nil
}

type MatrixItem struct {
	persistent bool
	op         uint8
	value      int64
	name       string
	result     string
}

func NewMatrixItem(op uint8, name string, value int64) *MatrixItem {
	return &MatrixItem{
		op:    op,
		name:  name,
		value: value,
	}
}

type MatrixScope struct {
	item *MatrixItem
	mode Mode
}

func NewMatrixScope(name string) MatrixScope {
	return MatrixScope{
		item: NewMatrixItem(kTimeDistribute, name, time.Now().UnixNano()/1000),
		mode: kModeLazy,
	}
}

func (m *MatrixScope) SetMode(mode Mode) {
	m.mode = mode
}

func (m *MatrixScope) SetOkay(result bool) {
	if result {
		m.item.result = "OK"
	} else {
		m.item.result = "ERR"
	}
}

func (m *MatrixScope) Scope() {
	m.item.value = time.Now().UnixNano()/1000 - m.item.value
	if m.item.result == "" && m.mode != kModeLazy {
		if m.mode == kModeAutoSuccess {
			m.item.result = "OK"
		} else {
			m.item.result = "ERR"
		}
	}
	SendToCollector(m.item)
}

const (
	kTotalUsec   int64 = 10 * 1000 * 1000
	kBucketUsec  int64 = 100
	kTotalBucket int64 = kTotalUsec / kBucketUsec
)

type MatrixState struct {
	persistent   bool
	start_sec    int64
	count        int64
	value        int64
	max          int64
	min          int64
	distribution []int64
	child        map[string]*MatrixState
}

func NewMatrixState(start int64, per bool) *MatrixState {
	return &MatrixState{
		start_sec:    start,
		persistent:   per,
		distribution: make([]int64, kTotalBucket),
		child:        make(map[string]*MatrixState),
	}
}

func (m *MatrixState) MaxMin(val int64) {
	m.max = util.Max(val, m.max)
	m.min = util.Min(val, m.min)
}

func (m *MatrixState) Set(val int64) {
	m.count += 1
	m.value = val
	m.MaxMin(val)
}

func (m *MatrixState) Add(val int64) {
	m.count += 1
	m.value += val
	m.MaxMin(val)
}

func (m *MatrixState) Sub(val int64) {
	m.count += 1
	m.value -= val
	m.MaxMin(val)
}

func (m *MatrixState) Reset() {
	m.value = 0
	m.count = 0
	m.min = 0
	m.max = 0xFFFFFFFF
}

func (m *MatrixState) GetQps() float64 {
	elapse := time.Now().UnixNano()/1000000 - m.start_sec
	if elapse <= 0 {
		elapse = 0
		return float64(0)
	}
	return float64(m.count / elapse)
}

func (m *MatrixState) GetAvg() float64 {
	if m.count == 0 {
		return float64(0)
	}
	return float64(m.value / m.count)
}

func (m *MatrixState) GetCount() int64 {
	return m.count
}

func (m *MatrixState) GetMax() int64 {
	return m.max
}

func (m *MatrixState) GetMin() int64 {
	return m.min
}

func (m *MatrixState) GetValue() int64 {
	return m.value
}

func (m *MatrixState) TimeDistribute(result string, value int64) {
	m.Distribute(value)
	if result == "" {
		return
	}
	if _, ok := m.child[result]; !ok {
		m.child[result] = NewMatrixState(m.start_sec, m.persistent)
	}
	v, _ := m.child[result]
	v.Distribute(value)
}

func (m *MatrixState) Distribute(value int64) {
	m.MaxMin(value)
	m.count += 1
	m.value += value
	bucket := value / kBucketUsec
	if bucket >= kTotalBucket {
		bucket = kTotalBucket - 1
	}
	m.distribution[bucket] += 1
}

func (m *MatrixState) GetTimeDistribute(percent float64) int64 {
	var (
		count     int64
		sum_count int64
		bucket    int64
	)
	count = int64(percent * float64(m.count))
	for ; bucket != int64(len(m.distribution)) && sum_count < count; bucket += 1 {
		sum_count += m.distribution[bucket]
	}
	if bucket != 0 {
		bucket -= 1
	}
	return bucket * kBucketUsec
}

func (m *MatrixState) Dump() string {
	s := "{\n\"qps\":" + strconv.FormatFloat(m.GetQps(), 'f', 3, 32) +
		",\n\"count\":" + strconv.FormatInt(m.GetCount(), 10) +
		",\n\"avg\":" + strconv.FormatFloat(m.GetAvg(), 'f', 3, 32) +
		",\n\"max\":" + strconv.FormatInt(m.GetMax(), 10) +
		",\n\"min\":" + strconv.FormatInt(m.GetMin(), 10)
	if len(m.distribution) != 0 {
		s += ",\n\"99\":" + strconv.FormatInt(m.GetTimeDistribute(0.99), 10) +
			",\n\"95\":" + strconv.FormatInt(m.GetTimeDistribute(0.95), 10) +
			",\n\"90\":" + strconv.FormatInt(m.GetTimeDistribute(0.90), 10) +
			",\n\"80\":" + strconv.FormatInt(m.GetTimeDistribute(0.80), 10) +
			",\n\"50\":" + strconv.FormatInt(m.GetTimeDistribute(0.50), 10)
	} else {
		s += ",\n\"value\":" + strconv.FormatInt(m.GetValue(), 10)
	}

	if len(m.child) != 0 {
		s += ",\n\"Result\":{"
		var (
			first bool = true
		)
		for k, v := range m.child {
			if first {
				first = false
				s += "\n			"
			} else {
				s += ",\n			"
			}
			s += "\"" + k + "\":" +
				v.Dump()
		}
		s += "\n}"
	}
	s += "\n}"
	return s
}

type MatrixStateMap struct {
	start_sec int64
	statMap   map[string]*MatrixState
}

func NewMatrixStateMap() *MatrixStateMap {
	return &MatrixStateMap{
		start_sec: time.Now().UnixNano() / 1000000,
		statMap:   make(map[string]*MatrixState),
	}
}

func (m *MatrixStateMap) SimpleString() string {
	simple := "{\n\"version:\" 1"
	for k, v := range m.statMap {
		simple += ",\n\"" + k + "\":" + v.Dump()
	}
	simple += "\n}"
	return simple
}

func (m *MatrixStateMap) GetStartTime() int64 {
	return m.start_sec
}

func (m *MatrixStateMap) Set(name string, value int64, persistent bool) {
	m.GetMatrixState(name, persistent).Set(value)
}

func (m *MatrixStateMap) Add(name string, value int64, persistent bool) {
	m.GetMatrixState(name, persistent).Add(value)
}

func (m *MatrixStateMap) Sub(name string, value int64, persistent bool) {
	m.GetMatrixState(name, persistent).Sub(value)
}

func (m *MatrixStateMap) Reset(name string) {
	m.GetMatrixState(name, false).Reset()
}

func (m *MatrixStateMap) Distribute(name string, value int64, persistent bool) {
	m.GetMatrixState(name, persistent).Distribute(value)
}

func (m *MatrixStateMap) TimeDistribute(name, result string, value int64, persistent bool) {
	m.GetMatrixState(name, persistent).TimeDistribute(result, value)
}

func (m *MatrixStateMap) GetMatrixState(name string, persistent bool) (stat *MatrixState) {
	var ok bool
	if stat, ok = m.statMap[name]; !ok {
		stat = NewMatrixState(m.start_sec, persistent)
		m.statMap[name] = stat
	}
	return stat
}

type MatrixCollector struct {
	queue    []*queue.Sequence
	bucket   uint32
	size     uint64
	running  uint64
	stat_map *MatrixStateMap
}

func NewMatrixCollector(bucket uint32, size uint64) *MatrixCollector {
	q := make([]*queue.Sequence, bucket)
	var i uint32
	for i = 0; i < bucket; i++ {
		q[i] = queue.NewSequence(size)
	}
	return &MatrixCollector{
		bucket: bucket,
		size:   size,
		queue:  q,
	}
}

func (m *MatrixCollector) Start() error {
	if !atomic.CompareAndSwapUint64(&m.running, 0, 1) {
		return errors.New("Started!")
	}
	smap := NewMatrixStateMap()
	go func() {
		for atomic.LoadUint64(&m.running) > 0 {
			if (time.Now().UnixNano()/1000000 - smap.GetStartTime()) > 30 {
				logger.Info("MatrixCollector:\n %v", smap.SimpleString())
				smap = NewMatrixStateMap()
			}
			var i uint32
			for i = 0; i != m.bucket; i++ {
				m.ProcessQueue(m.queue[i], smap)
			}
			time.Sleep(1000 * time.Millisecond)
		}
		logger.Info("MatrixCollector Exit")
	}()
	return nil
}

func (c *MatrixCollector) Send(item *MatrixItem) error {
	return c.queue[util.Goid()&int64(c.bucket-1)].Put(item)
}

func (m *MatrixCollector) ProcessQueue(q *queue.Sequence, sm *MatrixStateMap) {
	for {
		m, err := q.Get()
		if err != nil {
			return
		}
		item, ok := m.(*MatrixItem)
		if !ok {
			return
		}
		switch item.op {
		case kSet:
			sm.Set(item.name, item.value, item.persistent)
		case kAdd:
			sm.Add(item.name, item.value, item.persistent)
		case kSub:
			sm.Sub(item.name, item.value, item.persistent)
		case kReset:
			sm.Reset(item.name)
		case kTimeDistribute:
			if item.result == "" {
				sm.Distribute(item.name, item.value, item.persistent)
			} else {
				sm.TimeDistribute(item.name, item.result, item.value, item.persistent)
			}
		default:
			logger.Info("MatrixCollector.ProcessQueue Unknown Operation %v", item)
		}
	}
}
