package network

import (
	"barrage/logger"
	"barrage/msg"
	"barrage/protocol"
	"barrage/service"
	"net"
	"runtime"
	"time"
)

type Acceptor struct {
	io_service  *service.IOService
	service     msg.Service
	addr        string
	ln          *net.TCPListener
	proto       protocol.Protocol
	quit        chan struct{}
	freeChannel *ServiceChannel
}

func NewAcceptor(io_srv *service.IOService, srv msg.Service, address string, proto protocol.Protocol) *Acceptor {
	return &Acceptor{
		io_service: io_srv,
		service:    srv,
		addr:       address,
		proto:      proto,
		quit:       make(chan struct{}),
	}
}

func (a *Acceptor) Start() {
	defer a.ln.Close()
	netaddr, err := net.ResolveTCPAddr("tcp", a.addr)
	if err != nil {
		logger.Error("net.ResolveTCPAddr fail %v", netaddr)
		return
	}

	logger.Info("netaddr %v", netaddr)
	a.ln, err = net.ListenTCP("tcp", netaddr)
	if err != nil {
		logger.Error("net.ListenTCP fail")
		return
	}

	for i := 0; i < runtime.NumCPU(); i++ {
		go acceptTCP(a)
	}
}

func acceptTCP(a *Acceptor) {

	var (
		conn  *net.TCPConn
		delay time.Duration
		err   error
	)

	for {
		conn, err = a.ln.AcceptTCP()
		if err != nil {
			nerr, ok := err.(net.Error)
			if ok {
				if nerr.Timeout() {
					logger.Error("AcceptTCP timeout")
					return
				}
				if nerr.Temporary() {
					if delay == 0 {
						delay = 5 * time.Millisecond
					} else {
						delay *= 2
					}
					if max := 1 * time.Second; delay > max {
						delay = max
					}

					logger.Error("AcceptTCP Temporary error: %v,retry after : %v", err, delay)
					time.Sleep(delay)
					continue
				}
			}
			return
		}
		//conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		sch := a.allocChannel()
		InitChannel(sch, a, conn)
		sch.Start()
	}
}

func (a *Acceptor) Stop() {
	a.ln.Close()
	close(a.quit)
	Close()
}

func (a *Acceptor) allocChannel() *ServiceChannel {
	/*ch := a.freeChannel
	if ch == nil {
		ch = new(ServiceChannel)
	} else {
		a.freeChannel = ch.next
		*ch = ServiceChannel{}
	}
	return ch
	*/
	return &ServiceChannel{}
}
