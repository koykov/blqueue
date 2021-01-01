package queue

import (
	"encoding/json"
	"sync/atomic"
	"time"
)

const (
	defaultWakeupFactor = .75
	defaultSleepFactor  = .5
	defaultHeartbeat    = time.Millisecond

	spinlockLimit = 1000
)

type BalancedQueue struct {
	Queue

	workersUp int32
	acqlock   uint32
	spinlock  int64

	WakeupFactor float32
	SleepFactor  float32

	WorkersMin uint32
	WorkersMax uint32

	Heartbeat time.Duration
}

func (q *BalancedQueue) init() {
	if q.Metrics == nil {
		q.Metrics = &DummyMetrics{}
	}
	if q.Proc == nil {
		q.Proc = DummyProc
	}

	q.stream = make(stream, q.Size)

	if q.WorkersMin == 0 {
		q.WorkersMin = 1
	}
	if q.WorkersMax < q.WorkersMin {
		q.WorkersMax = q.WorkersMin
	}

	if q.WakeupFactor <= 0 {
		q.WakeupFactor = defaultWakeupFactor
	}
	if q.SleepFactor <= 0 {
		q.SleepFactor = defaultSleepFactor
	}
	if q.WakeupFactor < q.SleepFactor {
		q.WakeupFactor = q.SleepFactor
	}

	q.ctl = make([]ctl, q.WorkersMax)
	q.workers = make([]*worker, q.WorkersMax)
	var i uint32
	for i = 0; i < q.WorkersMax; i++ {
		q.Metrics.WorkerSleep(i)
		q.ctl[i] = make(chan signal)
		q.workers[i] = &worker{
			idx:     i,
			status:  wstatusIdle,
			proc:    q.Proc,
			metrics: q.Metrics,
		}
	}
	for i = 0; i < q.WorkersMin; i++ {
		go q.workers[i].observe(q.stream, q.ctl[i])
		q.ctl[i] <- signalInit
	}
	q.workersUp = int32(q.WorkersMin)

	if q.Heartbeat == 0 {
		q.Heartbeat = defaultHeartbeat
	}
	tickerHB := time.NewTicker(q.Heartbeat)
	go func() {
		for {
			select {
			case <-tickerHB.C:
				q.rebalance()
			}
		}
	}()

	q.status = qstatusActive
}

func (q *BalancedQueue) Put(x interface{}) bool {
	if q.status == qstatusNil {
		q.once.Do(q.init)
	}

	if atomic.AddInt64(&q.spinlock, 1) >= spinlockLimit {
		q.rebalance()
	}
	q.stream <- x
	q.Metrics.QueuePut()
	atomic.AddInt64(&q.spinlock, -1)
	return true
}

func (q *BalancedQueue) rebalance() {
	q.mux.Lock()
	defer q.mux.Unlock()
	if atomic.LoadUint32(&q.acqlock) == 1 {
		return
	}

	atomic.StoreUint32(&q.acqlock, 1)

	// Reset spinlock immediately to reduce amount of threads waiting for rebalance.
	q.spinlock = 0

	rate := q.lcRate()
	switch {
	case rate >= q.WakeupFactor:
		i := q.workersUp - 1
		go q.workers[i].observe(q.stream, q.ctl[i])
		q.ctl[i] <- signalResume
		atomic.AddInt32(&q.workersUp, 1)
	case rate <= q.SleepFactor:
		q.ctl[q.workersUp-1] <- signalSleep
		atomic.AddInt32(&q.workersUp, -1)
	case rate == 1:
		q.status = qstatusThrottle
	default:
		q.status = qstatusActive
	}

	atomic.StoreUint32(&q.acqlock, 0)
}

func (q *BalancedQueue) lcRate() float32 {
	return float32(len(q.stream)) / float32(cap(q.stream))
}

func (q *BalancedQueue) String() string {
	var out = &struct {
		Key           string  `json:"key"`
		Status        string  `json:"status"`
		Size          uint64  `json:"size"`
		WorkersMin    int     `json:"workers_min"`
		WorkersMax    int     `json:"workers_max"`
		WorkersIdle   int     `json:"workers_idle"`
		WorkersActive int     `json:"workers_active"`
		WorkersSleep  int     `json:"workers_sleep"`
		SleepFactor   float32 `json:"sleep_factor"`
		WakeupFactor  float32 `json:"wakeup_factor"`
	}{}
	out.Key = q.Key
	out.Size = q.Size

	switch q.status {
	case qstatusNil:
		out.Status = "inactive"
	case qstatusActive:
		out.Status = "active"
	case qstatusThrottle:
		out.Status = "throttle"
	}

	for _, w := range q.workers {
		if w == nil {
			out.WorkersIdle++
		} else {
			switch w.status {
			case wstatusIdle:
				out.WorkersIdle++
			case wstatusActive:
				out.WorkersActive++
			case w.status:
				out.WorkersSleep++
			}
		}
	}

	out.WorkersMin = int(q.WorkersMin)
	out.WorkersMax = int(q.WorkersMax)
	out.SleepFactor = q.SleepFactor
	out.WakeupFactor = q.WakeupFactor

	b, _ := json.Marshal(out)

	return string(b)
}
