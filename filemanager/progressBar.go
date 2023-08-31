package filemanager

import (
	"fmt"
	"io"
	"sync"
	"time"
)

type CountingReader struct {
	Reader io.Reader
	Count  int64
	mu     sync.Mutex // 用于保护 Count
}

func (cr *CountingReader) Read(p []byte) (n int, err error) {
	n, err = cr.Reader.Read(p)
	cr.mu.Lock()
	cr.Count += int64(n)
	cr.mu.Unlock()
	return n, err
}

func ProgressBar(cr *CountingReader, finishChan chan struct{}) {
	tick := time.Tick(100 * time.Millisecond)
	finish := false
	for {
		if finish {
			break
		}
		select {
		case <-tick:
			cr.mu.Lock()
			count := cr.Count
			cr.mu.Unlock()
			// 更新进度条（或者其它逻辑）
			fmt.Printf("\rTotal bytes read from stdin: %d\n", count)
		case <-finishChan:
			cr.mu.Lock()
			count := cr.Count
			cr.mu.Unlock()
			// 更新进度条（或者其它逻辑）
			fmt.Printf("\rTotal bytes read from stdin: %d\n", count)
			finish = true
		}
	}
}
