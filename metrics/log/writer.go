package log

import (
	"log"

	"github.com/koykov/blqueue"
)

type Log struct {
	queue string
}

func NewMetricsWriter(queueKey string) *Log {
	m := &Log{queue: queueKey}
	return m
}

func (m *Log) WorkerSetup(active, sleep, stop uint) {
	log.Printf("queue #%s: setup workers %d active, %d sleep and %d stop", m.queue, active, sleep, stop)
}

func (m *Log) WorkerInit(idx uint32) {
	log.Printf("queue %s: worker %d caught init signal\n", m.queue, idx)
}

func (m *Log) WorkerSleep(idx uint32) {
	log.Printf("queue %s: worker %d caught sleep signal\n", m.queue, idx)
}

func (m *Log) WorkerWakeup(idx uint32) {
	log.Printf("queue %s: worker %d caught wakeup signal\n", m.queue, idx)
}

func (m *Log) WorkerStop(idx uint32, force bool, status blqueue.WorkerStatus) {
	if force {
		log.Printf("queue %s: worker %d caught force stop signal (current status %d)\n", m.queue, idx, status)
	} else {
		log.Printf("queue %s: worker %d caught stop signal\n", m.queue, idx)
	}
}

func (m *Log) QueuePut() {
	log.Printf("queue %s: new item come to the queue\n", m.queue)
}

func (m *Log) QueuePull() {
	log.Printf("queue %s: item leave the queue\n", m.queue)
}

func (m *Log) QueueLeak() {
	log.Printf("queue %s: queue leak\n", m.queue)
}

func (m *Log) QueueClose() {
	log.Printf("queue %s: queue close\n", m.queue)
}
