package network

import (
	"io"
	"net"
	"time"

	"barrage/logger"
	"barrage/msg"
	"barrage/protocol"
	"barrage/queue"
)

const (
	per_read_size    = 4096
	decode_watermask = 4096
)

type ServiceChannel struct {
	in       *queue.IOBuffer
	out      *queue.IOBuffer
	conn     *net.TCPConn
	attrs    map[string]string
	acceptor *Acceptor
	next     *ServiceChannel
	queue    chan msg.Message
}

func InitChannel(sch *ServiceChannel, a *Acceptor, c *net.TCPConn) {
	sch.in = queue.NewIOBuffer(true)
	sch.out = queue.NewIOBuffer(false)
	sch.conn = c
	sch.acceptor = a
	sch.attrs = make(map[string]string)
	sch.queue = make(chan msg.Message,1024)
}

func NewChannel() *ServiceChannel {
	return &ServiceChannel{
		attrs: make(map[string]string),
	}
}

func (s *ServiceChannel) SetAttr(key, value string) {
	s.attrs[key] = value
}

func (s *ServiceChannel) GetAttr(key string) string {
	if attr, ok := s.attrs[key]; ok {
		return attr
	}
	return ""
}

func (s *ServiceChannel) Start() {
	go s.OnRead()
	go s.OnWrite()
}

func (s *ServiceChannel) OnRead() {
	var (
		err error
		wt  uint64
		n   int
	)
	defer func() {
		if s.GetAttr("status") == "OK" {
			logger.Info(" %v leave room %v", s.GetAttr("cid"), s.GetAttr("rid"))
			UnRegister(s.GetAttr("cid"), s.GetAttr("uid"), s.GetAttr("rid"))
			NotifyHost(s.GetAttr("rid"), s.GetAttr("cid"), s.GetAttr("uid"), 0)
		}
		s.in.Reset()
		s.out.Reset()
		s.conn.Close()
	}()

L:
	for {
		select {
		case <-s.acceptor.quit:
			break L
		default:
		}
		wt, err = s.in.EnsureWrite(per_read_size)
		if err != nil {
			logger.Error("EnsureWrite %v", err)
			break L
		}

		n, err = s.conn.Read(s.in.Buffer()[wt:])
		if err != nil {
			if err == io.EOF {
				logger.Warn("ServiceChannel.OnRead Closed by peer %v", err)
			} else {
				logger.Error("ServiceChannel.OnRead Error %v", err)
			}
			break L
		}
		s.in.Produce(uint64(n))
		err = s.DecodeMessage()	
		if err != nil {
			logger.Info("Decode error %v", err)
			if err == protocol.HeaderErr || err == protocol.BodyErr {
				continue
			}

			break L
		}
	}
}

func (s *ServiceChannel) OnWrite() {
	for {
		select {
		case <-s.acceptor.quit:
			close(s.queue)
			return
		default:
			m := <-s.queue
			s.EncodeMessage(m)
		}
	}
}

func (s *ServiceChannel) Write() {
	//for s.out.GetReadSize() > 0 {
	if s.out.GetRead() > s.out.GetWrite() {
		return
	}

	n, err := s.conn.Write(s.out.Buffer()[s.out.GetRead():s.out.GetWrite()])
	if n < 0 || err != nil {
		return
	}
	s.out.Consume(uint64(n))
	//}
}

func (s *ServiceChannel) Read() {}
func (s *ServiceChannel) DecodeMessage() error {
	var (
		m msg.Message
		e error
	)
	for s.in.GetReadSize() > 0 && e == nil {
		m, e = s.acceptor.proto.Decode(s.in)
		if e != nil {
			return e
		}
		m.SetChannel(s)
		//logger.Info("decode msg %s",m.(*protocol.Barrage).Body)
		//s.acceptor.io_service.GetServiceStage().Send(m)
		s.Serve(m)
		s.in.SetInit(true)
	}
	return nil
}

func (s *ServiceChannel) EncodeMessage(msg msg.Message) {
	if err := s.acceptor.proto.Encode(msg, s.out); err != nil {
		logger.Error("s.protocol.Encode error %v", err)
		s.conn.Close()
		return
	}
	s.Write()
}

func (s *ServiceChannel) SendIO(msg msg.Message) {
	s.acceptor.io_service.GetIOStage().Send(msg)
}

func (s *ServiceChannel) Send(msg msg.Message) {
	/*select {
	case <-s.acceptor.quit:
		return
	case s.queue <- msg:
	}*/
	s.queue <- msg
}

func (s *ServiceChannel) Serve(msg msg.Message) {
	s.acceptor.service.Serve(msg)
}

func (s *ServiceChannel) GetIOService() msg.Service {
	return s.acceptor.io_service
}

func (s *ServiceChannel) SetDeadline(timestamp int) {
	s.conn.SetDeadline(time.Now().Add(time.Duration(timestamp) * time.Second))
}

func (s *ServiceChannel) OnClose() {}

func (s *ServiceChannel) Close() {
	s.conn.Close()
}
