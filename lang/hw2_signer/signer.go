package main

// сюда писать код
import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func ExecutePipeline(jobs ...job) {
	in := make(chan interface{})
	wg := &sync.WaitGroup{}
	for i, pipeJob := range jobs {
		out := make(chan interface{})
		wg.Add(1)
		go func(in, out chan interface{}, pipe job, i int) {
			defer func() {
				close(out)
				wg.Done()
			}()
			pipe(in, out)
		}(in, out, pipeJob, i)
		in = out
	}
	wg.Wait()
}

type ordered struct {
	num  int
	data string
}

func SingleHash(in chan interface{}, out chan interface{}) {
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	for val := range in {
		data := fmt.Sprintf("%v", val)

		wg.Add(1)
		go func(data string) {
			defer wg.Done()

			inCrc32 := make(chan ordered)
			outCrc32 := make(chan ordered)

			for i := 0; i < 2; i++ {
				go func() {
					crc32Data := <-inCrc32
					crc32Res := DataSignerCrc32(crc32Data.data)
					outCrc32 <- ordered{crc32Data.num, crc32Res}
				}()
			}
			inCrc32 <- ordered{0, data}

			go func(data string, outMd5 chan<- ordered) {
				mu.Lock()
				fromMd5Res := ordered{1, DataSignerMd5(data)}
				mu.Unlock()
				outMd5 <- fromMd5Res
			}(data, inCrc32)

			h1 := <-outCrc32
			h2 := <-outCrc32
			if h1.num < h2.num {
				out <- h1.data + "~" + h2.data
			} else {
				out <- h2.data + "~" + h1.data
			}
		}(data)
	}
	wg.Wait()
}

func MultiHash(in chan interface{}, out chan interface{}) {
	wg := &sync.WaitGroup{}
	for val := range in {
		data := fmt.Sprintf("%v", val)

		wg.Add(1)
		go func(data string) {
			defer wg.Done()

			inCrc32 := make(chan ordered)
			outCrc32 := make(chan ordered)

			for i := 0; i <= 5; i++ {
				go func() {
					crc32Data := <-inCrc32
					crc32Res := DataSignerCrc32(strconv.Itoa(crc32Data.num) + crc32Data.data)
					outCrc32 <- ordered{crc32Data.num, crc32Res}
				}()
				inCrc32 <- ordered{i, data}
			}

			result := make(map[int]string)
			for i := 0; i <= 5; i++ {
				crc32Res := <-outCrc32
				result[crc32Res.num] = crc32Res.data
			}

			hash := ""
			for i := 0; i <= 5; i++ {
				hash = hash + result[i]
			}

			out <- hash
		}(data)
	}
	wg.Wait()
}

func CombineResults(in chan interface{}, out chan interface{}) {
	results := make([]string, 0)
	for val := range in {
		results = append(results, fmt.Sprintf("%v", val))
	}
	sort.Strings(results)
	out <- strings.Join(results, "_")
}
