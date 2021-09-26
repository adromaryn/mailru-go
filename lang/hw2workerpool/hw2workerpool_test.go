package hw2workerpool

import (
	"fmt"
	"testing"
	"time"
)

func TestWorkerPoolOnAdd(t *testing.T) {
	jobs := make(chan func(chan interface{}), 20)
	results := make(chan interface{}, 20)
	fmt.Println("create worker pool on add")
	wp := StartWorkerPool(3, jobs, results)
	if wp.Size() != 3 {
		t.Errorf("capacity of initialized worker pool wrong: %v", wp.Size())
	}
	if wp.ActiveCount() != 0 {
		t.Errorf("length of initialized worker pool should be zero: %v", wp.ActiveCount())
	}

	fmt.Println("start jobs")
	for i := 0; i < 5; i++ {
		jobs <- func(results chan interface{}) {
			fmt.Println("start")
			time.Sleep(time.Second * 4)
			results <- struct{}{}
			fmt.Println("stop")
		}
	}
	time.Sleep(time.Second)
	if wp.Size() != 3 {
		t.Errorf("capacity of started worker pool wrong: %v", wp.Size())
	}
	if wp.ActiveCount() != 3 {
		t.Errorf("length of started worker pool wrong: %v", wp.ActiveCount())
	}

	fmt.Println("add workers")
	wp.AddWorkers(3)
	time.Sleep(time.Second * 2)
	fmt.Println("all jobs in process")
	if wp.Size() != 6 {
		t.Errorf("capacity of extended worker pool wrong: %v", wp.Size())
	}
	if wp.ActiveCount() != 5 {
		t.Errorf("length of extended worker pool wrong: %v", wp.ActiveCount())
	}

	time.Sleep(time.Second * 4)
	fmt.Println("all jobs processed")
	if wp.ActiveCount() != 0 {
		t.Errorf("length of done worker pool wrong: %v", wp.ActiveCount())
	}

	wp.Finish()
}

func TestWorkerPoolOnDec(t *testing.T) {
	jobs := make(chan func(chan interface{}), 20)
	results := make(chan interface{}, 20)
	fmt.Println("create worker pool on dec")
	wp := StartWorkerPool(5, jobs, results)
	if wp.Size() != 5 {
		t.Errorf("capacity of initialized worker pool wrong: %v", wp.Size())
	}
	if wp.ActiveCount() != 0 {
		t.Errorf("length of initialized worker pool should be zero: %v", wp.ActiveCount())
	}

	fmt.Println("start jobs")
	for i := 0; i < 8; i++ {
		jobs <- func(results chan interface{}) {
			time.Sleep(time.Second * 3)
			results <- struct{}{}
		}
	}
	time.Sleep(time.Second)
	fmt.Println("first 5 jobs processing")
	if wp.Size() != 5 {
		t.Errorf("capacity of started worker pool wrong: %v", wp.Size())
	}
	if wp.ActiveCount() != 5 {
		t.Errorf("length of started worker pool wrong: %v", wp.ActiveCount())
	}

	fmt.Println("dec workers")
	wp.DecWorkers(3)
	time.Sleep(time.Second * 4)
	fmt.Println("next 2 jobs processing")
	if wp.Size() != 2 {
		t.Errorf("capacity of decremented worker pool wrong: %v", wp.Size())
	}
	if wp.ActiveCount() != 2 {
		t.Errorf("length of decremented worker pool wrong: %v", wp.ActiveCount())
	}

	time.Sleep(time.Second * 4)
	fmt.Println("last job processing")
	if wp.ActiveCount() != 1 {
		t.Errorf("length of done worker pool wrong: %v", wp.ActiveCount())
	}
	wp.Finish()
}
