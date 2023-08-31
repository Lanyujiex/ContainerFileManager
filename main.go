package main

import (
	"fmt"
	"log"
	"myfile.manager/filemanager"
	"net/http"
)

func main() {
	//fmt.Println("wo yao kaishi le ")
	// 设置路由处理函数
	http.HandleFunc("/", homeHandler)
	fileManager := filemanager.FileManager{}

	http.HandleFunc("/upload", fileManager.UploadHandler)
	http.HandleFunc("/download", fileManager.DownloadHandler)

	// 启动 HTTP 服务器，监听在本地的 8080 端口
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "这是一个测试的Web服务器!")
}
