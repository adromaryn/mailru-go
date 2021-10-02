package hw2workerpool

import (
	"context"
	"errors"
	"sync"
)

type WorkerPool struct {
	bufChan chan struct{}
	resync  chan struct{}
	jobs    chan func(results chan interface{})
	results chan interface{}
	ctx     context.Context
	finish  context.CancelFunc
	lock    *sync.Mutex
	counter int
}

func StartWorkerPool(count int, jobs chan func(results chan interface{}), results chan interface{}) *WorkerPool {
	bufChan := make(chan struct{}, count)
	resync := make(chan struct{})
	ctx, finish := context.WithCancel(context.Background())
	lock := &sync.Mutex{}
	wp := &WorkerPool{bufChan, resync, jobs, results, ctx, finish, lock, 0}

	go func(wp *WorkerPool) {
		for {
			wp.lock.Lock()
			select {
			case <-ctx.Done():
				return
			default:
			}
			bufChan := wp.bufChan
			resync := wp.resync
			ctx := wp.ctx
			select {
			case bufChan <- struct{}{}:
				select {
				case j := <-jobs:
					go func() {
						wp.lock.Lock()
						wp.counter = wp.counter + 1
						wp.lock.Unlock()

						defer func() {
							wp.lock.Lock()
							select {
							case <-resync:
								if wp.counter <= cap(wp.bufChan) {
									<-wp.bufChan
								}
							default:
								<-bufChan
							}
							wp.counter = wp.counter - 1
							wp.lock.Unlock()
						}()

						j(wp.results)
					}()
				case <-ctx.Done():
					wp.lock.Unlock()
					return
				default:
					<-bufChan
				}
			case <-ctx.Done():
				wp.lock.Unlock()
				return
			default:
			}
			wp.lock.Unlock()
		}
	}(wp)

	return wp
}

func (wp *WorkerPool) Finish() {
	wp.lock.Lock()
	defer wp.lock.Unlock()
	wp.finish()
	close(wp.bufChan)
}

func (wp *WorkerPool) AddWorkers(count int) {
	wp.lock.Lock()
	defer wp.lock.Unlock()
	currentCap := cap(wp.bufChan)
	currentLen := len(wp.bufChan)
	newBufChan := make(chan struct{}, currentCap+count)
	close(wp.bufChan)
	wp.bufChan = newBufChan
	for i := 0; i < currentLen; i++ {
		newBufChan <- struct{}{}
	}
}

func (wp *WorkerPool) DecWorkers(count int) error {
	wp.lock.Lock()
	defer wp.lock.Unlock()
	currentCap := cap(wp.bufChan)
	if currentCap <= count {
		return errors.New("tried to stop all workers")
	}
	currentLen := len(wp.bufChan)
	currentChan := wp.bufChan
	newBufChan := make(chan struct{}, currentCap-count)
	wp.bufChan = newBufChan
	if currentLen < currentCap-count {
		for i := 0; i < currentLen; i++ {
			newBufChan <- struct{}{}
		}
	} else {
		for i := 0; i < currentCap-count; i++ {
			newBufChan <- struct{}{}
		}
	}
	close(wp.resync)
	close(currentChan)
	wp.resync = make(chan struct{})

	return nil
}

func (wp *WorkerPool) Size() int {
	wp.lock.Lock()
	defer wp.lock.Unlock()
	count := cap(wp.bufChan)
	return count
}

func (wp *WorkerPool) ActiveCount() int {
	wp.lock.Lock()
	defer wp.lock.Unlock()
	return wp.counter
}
