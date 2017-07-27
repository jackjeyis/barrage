package service

import (
	"goio/msg"
	"sync"
)

type Worker struct {
	queue   *queue.Sequence
	running bool
	wg      *sync.WaitGroup
}

func NewWorker(qs int) *Worker {
	return &Worker{
		running: true,
		wg:      &sync.WaitGroup{},
		queue:   queue.NewSequece(uint64(qs)),
	}
}

func (w *Worker) Run(h Handler) {
	w.wg.Add(1)
	go func() {
		for w.running {
			m, err := queue.Get()
			if err != nil {
				continue
			}
			h(m)
		}
		w.wg.Done()
	}()
}

func (w *Worker) Wait() {
	w.wg.Wait()
}

func (w *Worker) Stop() {
	w.running = false
}

type Boss struct {
	workers     []*Worker
	workerCount int
	queueSize   int
}

func NewBoss(wc, qs int, hander Handler) *Boss {
	return &Boss{
		workerCount: wc,
		queueSize:   qs,
	}
}

func (s *Boss) Start(h Handler) {
	for i := 0; i < s.workerCount; i++ {
		worker := NewWorker(s.qs)
		s.workers[i] = worker
		h.SetHandlerId(i)
		worker.Run(h)
	}
}

func (s *Boss) Wait() {
	for i := 0; i < s.workerCount; i++ {
		s.workers[i].Wait()
	}
}

func (s *Boss) Stop() {
	for i := 0; i < s.workerCount; i++ {
		s.workers[i].Stop()
	}
}

func (s *Boss) Send(m msg.Message) {
	s.works[m.HandlerId()%s.workerCount].queue.Put(m)
}
