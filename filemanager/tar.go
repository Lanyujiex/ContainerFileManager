package filemanager

import (
	"archive/tar"
	"io"
	"net/http"
)

func createTarStream(r *http.Request) (io.Reader, error) {
	// 创建管道
	pipeReader, pipeWriter := io.Pipe()

	// 使用goroutine在后台生成tar文件流
	go func() {
		tarWriter := tar.NewWriter(pipeWriter)
		defer func() {
			tarWriter.Close()
			pipeWriter.Close()
		}()

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

				// 创建tar条目
				tarHeader := &tar.Header{
					Name: header.Filename,
					Size: header.Size,
				}
				err = tarWriter.WriteHeader(tarHeader)
				if err != nil {
					pipeWriter.CloseWithError(err)
					return
				}

				// 写入文件内容到tar流
				_, err = io.Copy(tarWriter, file)
				if err != nil {
					pipeWriter.CloseWithError(err)
					return
				}
			}
		}
	}()

	return pipeReader, nil
}
