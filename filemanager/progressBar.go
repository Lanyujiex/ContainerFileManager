package filemanager

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// 重定义一个IOReader，用来记录读取的字节书
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

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var progressChanCache = map[string]chan int64{}

var proChanLock sync.RWMutex

func getKey(containerName, podName, namespace, filePath, fileName, opId string) string {
	key := fmt.Sprintf("%s-%s-%s-%s-%s-%s", containerName, podName, namespace, filePath, fileName, opId)
	return key
}

func ProgressBar(cr *CountingReader, finishChan chan struct{}, key string) {
	tick := time.Tick(500 * time.Millisecond)
	finish := false

	curChan := GetProgressChan(key)
	for {
		if finish {
			close(curChan)
			break
		}
		select {
		case <-tick:
			cr.mu.Lock()
			count := cr.Count
			cr.mu.Unlock()
			// 更新进度条（或者其它逻辑）
			fmt.Printf("\rTotal bytes read from stdin: %d\n", count)
			curChan <- count
		case <-finishChan:
			cr.mu.Lock()
			count := cr.Count
			cr.mu.Unlock()
			// 更新进度条（或者其它逻辑）
			fmt.Printf("\rTotal bytes read from stdin: %d\n\rUpload file finished, key=%s\n", count, key)
			curChan <- count
			finish = true
		}
	}
}

func ProgressSocket(w http.ResponseWriter, r *http.Request) {
	namespace := r.URL.Query().Get("namespace")
	podName := r.URL.Query().Get("podName")
	containerName := r.URL.Query().Get("containerName")
	fileName := r.URL.Query().Get("fileName")
	filePath := r.URL.Query().Get("filePath")
	opId := r.URL.Query().Get("opId")
	key := getKey(containerName, podName, namespace, filePath, fileName, opId)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading to WebSocket:", err)
		return
	}
	defer conn.Close()
	fmt.Println("Client connected")

	//messageType, msg, err := conn.ReadMessage()
	//if err != nil {
	//	fmt.Println("Error reading message:", err)
	//}
	//fmt.Println(messageType, "   ", msg)

	proChan := GetProgressChan(key)
	initMsg := fmt.Sprintf("upload init")
	fmt.Println(initMsg)
	err = conn.WriteMessage(1, []byte(initMsg))
	if err != nil {
		fmt.Println("Error writing message:", err)
		return
	}
	finish := false

	for {
		if finish {
			break
		}

		select {
		case t, ok := <-proChan:
			if !ok {
				finish = true
				fmt.Printf("remove chan from cache map key=%s \n", key)
				DeleteProgressChan(key)
				break
			}
			proMsg := fmt.Sprintf("send %v Bytes", t)
			err = conn.WriteMessage(1, []byte(proMsg))
			if err != nil {
				fmt.Println("Error writing message:", err)
				break
			}
		}
	}
}

func GetProgressChan(key string) chan int64 {
	if progressChanCache == nil {
		progressChanCache = map[string]chan int64{}
	}
	proChanLock.RLock()
	opChan, ok := progressChanCache[key]
	if ok {
		proChanLock.RUnlock()
		return opChan
	}
	proChanLock.RUnlock()

	//避免读写锁冲突
	proChanLock.Lock()
	defer proChanLock.Unlock()
	opChan = make(chan int64, 1)
	progressChanCache[key] = opChan
	return opChan
}

func DeleteProgressChan(key string) {
	proChanLock.Lock()
	defer proChanLock.Unlock()
	delete(progressChanCache, key)
}
