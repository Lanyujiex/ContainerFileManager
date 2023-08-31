package filemanager

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func createIOStream(r *http.Request) (io.Reader, error) {
	// 创建管道
	pipeReader, pipeWriter := io.Pipe()

	go func() {
		// 遍历所有文件字段
		err := r.ParseMultipartForm(32 << 20) // 解析32MB的最大multipart/form-data请求
		if err != nil {
			pipeWriter.CloseWithError(err)
			return
		}
		for _, headers := range r.MultipartForm.File {
			for _, header := range headers {
				// 打开上传的文件
				file, err := header.Open()
				if err != nil {
					pipeWriter.CloseWithError(err)
					return
				}
				defer file.Close()
				defer pipeWriter.Close()
				_, err = io.Copy(pipeWriter, file)
				if err != nil {
					// 错误处理
					fmt.Println(err)
				}
			}
		}
	}()

	return pipeReader, nil
}

func createFile(fileName string, reader *io.Reader) error {
	file, err := os.Create(fileName)
	if err != nil {
		// 错误处理
		fmt.Println(err)
	}
	defer file.Close()

	// 将 tarStream 写入文件
	_, err = io.Copy(file, *reader)
	if err != nil {
		fmt.Println(err)
	}
	return nil
}
