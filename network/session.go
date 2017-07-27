package network

import (
	"barrage/logger"
	"barrage/msg"
	"barrage/protocol"
	"barrage/util"
	"sync/atomic"
)

var (
	b  *Bucket
	q  chan msg.Message
	l uint64
	bucketq []chan msg.Message
)

func init() {
	once.Do(func() {
		q = make(chan msg.Message, 1024)
		b = NewBucket(256)
		bucketq = make([]chan msg.Message,1024)
		for i,_ := range bucketq {
			bucketq[i] = make(chan msg.Message,128)
			go process(bucketq[i])
		}
	})
}


func process(q chan msg.Message) {
	for {
		m := <-q
		for _,s := range b.buckets {
			s.pushMsg(m)
		}
	}
}

type Bucket struct {
	buckets []*Session
	size    uint32
}

func NewBucket(size uint32) *Bucket {
	b := &Bucket{
		buckets: make([]*Session, size),
		size:    size,
	}

	for i := uint32(0); i < size; i++ {
		b.buckets[i] = NewSession()
	}

	return b
}

func (b *Bucket) HashSession(key string) *Session {
	idx := util.CityHash32([]byte(key), uint32(len(key))) % b.size
	return b.buckets[idx]
}

type Session struct {
	ctrie *util.Map
	room  *util.Map
}

func NewSession() *Session {

	s := &Session{
		ctrie: &util.Map{},
		room:  &util.Map{},
	}
	return s
}

func (s *Session) pushMsg(m msg.Message) {
	r := s.roomer(m.Channel().GetAttr("rid"))
	if r == nil {
		return
	}
	r.pushMsg(m)
}

func (s *Session) pushHost(m msg.Message) {
	r := s.roomer(m.Channel().GetAttr("rid"))
	if r == nil {
		return
	}
		r.host.Range(func(key, value interface{}) bool {
			c := value.(msg.Channel)
			if c == nil || c.GetAttr("cid") == m.Channel().GetAttr("cid") {
				return true
			}
			c.Send(m)
			return true
		})
		
}

type Room struct {
	chs  *util.Map
	rid  string
	host *util.Map
}

func NewRoom(id string) *Room {
	r := &Room{
		chs:  &util.Map{},
		rid:  id,
		host: &util.Map{},
	}
	go r.handle()
	return r
}

func BroadcastHost(m msg.Message) {
	for _,s := range b.buckets {
		s.pushHost(m)
	}	
}

func (r *Room) handleLeave() {
	for {
		b := <-q
		if b == nil {
			return
		}
		b.Channel().Send(b)
	}
}

func (r *Room) pushMsg(m msg.Message) {
	r.chs.Range(func(key, value interface{}) bool {
		c := value.(msg.Channel)
		if c == nil || c.GetAttr("cid") == m.Channel().GetAttr("cid") {
			return true
		}
		/*b := &protocol.Barrage{}
		b.SetChannel(c)
		b.Op = 5
		b.Ver = 1
		b.Body = m.(*protocol.Barrage).Body
		c.SendIO(b)
		*/
		c.Send(m)
		return true
	})
}

func (r *Room) handle() {
	r.handleLeave()
}

func (r *Room) getMember() (uids []string, rcount int) {
	r.chs.Range(func(key, value interface{}) bool {
		uids = append(uids, key.(string))
		rcount += 1
		return true
	})
	return
}

func (r *Room) pushHost(b *protocol.Barrage) {
	q <- b
}

func (r *Room) addChan(uid string, ch msg.Channel) {
	r.chs.Store(uid, ch)
}

func (r *Room) deleteChan(cid, uid string) {
	r.chs.Delete(uid)
	if r.chs.Size() == 0 {
		b.HashSession(cid).room.Delete(r.rid)
	}
}

func (s *Session) insert(cid string, ch msg.Channel, host int) {
	var room *Room
	s.ctrie.Store(cid, ch)
	rid := ch.GetAttr("rid")
	r, _ := s.room.Load(rid)
	if r == nil {
		room = NewRoom(rid)
		s.room.Store(rid, room)
	} else {
		room = r.(*Room)
	}
	if host != 2 {
		room.host.Store(cid, ch)
	}
	room.addChan(ch.GetAttr("uid"), ch)
}

func (s *Session) delete(cid, uid, rid string) {
	s.ctrie.Delete(cid)
	r := s.roomer(rid)
	if r != nil {
		r.deleteChan(cid, uid)
		r.host.Delete(cid)
	}
}

func (s *Session) find(cid string) msg.Channel {
	ch, ok := s.ctrie.Load(cid)
	if ok {
		return ch.(msg.Channel)
	}
	return nil
}

func (s *Session) roomer(rid string) *Room {
	r, ok := s.room.Load(rid)
	if !ok {
		return nil
	}
	return r.(*Room)
}

func Register(cid string, ch msg.Channel, host int) {
	b.HashSession(cid).insert(cid, ch, host)
}

func UnRegister(cid, uid, rid string) {
	b.HashSession(cid).delete(cid, uid, rid)
}

func GetSession(cid string) msg.Channel {
	return b.HashSession(cid).find(cid)
}

func GetRoomStatus(rid string) (rn int, uids []string) {
	for _, s := range b.buckets {
		r := s.roomer(rid)
		if r == nil {
			continue
		}
		us, count := r.getMember()
		rn += count
		uids = append(uids, us...)
	}
	return rn, uids
}

func NotifyHost(rid, cid, uid string, code int8) {
	notify := protocol.Notify{}
	notify.Id = util.UUID()
	notify.Ct = 90010
	notify.Uid = uid
	notify.Rid = rid
	notify.Code = code
	notify.Time = util.GetMillis()
	body, err := util.EncodeJson(notify)
	//err = util.StoreMessage("http://"+util.GetHttpConfig().Remoteaddr+"/im/"+rid+"/view_record", body)

	if err != nil {
		logger.Error("util.StoreMessage error %v", err)
	}
	b := &protocol.Barrage{}
	b.Body = body
	b.Op = 5
	b.Ver = 1
	ch := NewChannel()
	ch.SetAttr("rid", rid)
	ch.SetAttr("cid", cid)
	b.SetChannel(ch)
	if code == 0 {
		//BroadcastHost(b)
	} else {
		BroadcastRoom(b, false)
	}
}

func BroadcastRoom(m msg.Message, store bool) {
	/*if store {
		err := util.StoreMessage("http://"+util.GetHttpConfig().Remoteaddr+"/im/"+barrage.Channel().GetAttr("rid")+"/chat", barrage.Body)

		if err != nil {
			logger.Error("util.StoreMessage error %v", err)
		}
	}*/
	idx := atomic.AddUint64(&l,1) % 1024
	bucketq[idx] <- m
}

func Close() {
	/*for _,s := range b.buckets {
		for i := uint64(0); i < s.msize; i++ {
			close(s.msgq[i])
		}
	}*/
	close(q)
}
