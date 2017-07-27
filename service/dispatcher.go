package service

import (
	"barrage/msg"
	"sync"
	"sync/atomic"
	"barrage/queue"
)

const (
	WORKER_STARTED = iota
	WORKER_STOPPED
	DISPATHER_STARTED
	DISPATHER_STOPPED
)

type Worker struct {
	workerPool chan chan msg.Message
	jobChannel chan msg.Message
	quit       chan struct{}
	handler    Handler
	wg         *sync.WaitGroup
	running    bool
	id         int
	status     byte
	queue *queue.Sequence
}

type Dispatcher struct {
	queue      chan msg.Message
	workerPool chan chan msg.Message
	maxWorkers int
	workers    []*Worker
	status     byte
	stop       chan struct{}
	cn uint64
}

func NewWorker(i int, h Handler, pool chan chan msg.Message) *Worker {
	return &Worker{
		workerPool: pool,
		handler:    h,
		jobChannel: make(chan msg.Message),
		wg:         &sync.WaitGroup{},
		id:         i,
		running:    true,
		quit:       make(chan struct{}),
		queue:queue.NewSequence(16),
	}
}

func (w *Worker) Start() {
	w.wg.Add(1)
	w.status = WORKER_STARTED
	go func() {
		for w.running {
			//w.workerPool <- w.jobChannel
			m := <-w.jobChannel
			//m, _ := w.queue.Get()
			if m != nil {
				w.handler.Handle(m)
			}
		}
		w.wg.Done()
	}()
}

func (w *Worker) Stop() {
	w.running = false
	close(w.jobChannel)
}

func (w *Worker) Wait() {
	w.wg.Wait()
}

func NewDispatcher(maxWorkers, maxQueue int) *Dispatcher {
	return &Dispatcher{
		workerPool: make(chan chan msg.Message, maxWorkers),
		queue:      make(chan msg.Message, maxQueue),
		maxWorkers: maxWorkers,
		workers:    make([]*Worker, maxWorkers),
		stop:       make(chan struct{}),
	}
}

func (d *Dispatcher) Run(h Handler) {
	for i := 0; i < d.maxWorkers; i++ {
		worker := NewWorker(i, h, d.workerPool)
		worker.Start()
		d.workers[i] = worker
	}
	//go d.Dispatch()
}

func (d *Dispatcher) Stop() {
	close(d.stop)
	for _, worker := range d.workers {
		worker.Stop()
	}
	close(d.workerPool)
}

func (d *Dispatcher) Wait() {
	for _, worker := range d.workers {
		worker.Wait()
	}
}

func (d *Dispatcher) Dispatch() {
	for {
		select {
		case m := <-d.queue:
			if m != nil {
				go func(msg msg.Message) {
					jobChannel := <-d.workerPool
					if jobChannel != nil {
						jobChannel <- m
					}
				}(m)
			}
		case <-d.stop:
			return
		}
	}
}

func (d *Dispatcher) Send(msg msg.Message) {
	idx := atomic.AddUint64(&d.cn,1) % uint64(d.maxWorkers)
	d.workers[idx].jobChannel <- msg
	//d.workers[idx].queue.Put(msg)
	//d.workers[msg.HandlerId()%d.maxWorkers].jobChannel <- msg
	//d.queue <- msg
}
