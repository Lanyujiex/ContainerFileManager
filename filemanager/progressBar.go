package filemanager

import (
	"fmt"
	"io"
	"os"
)

func copyWithProgressBar(dst io.Writer, src io.Reader, size int64) (int64, error) {
	//progressBar := pb.Full.Start64(size)
	//progressBar.Set(pb.Bytes, true)
	//
	//n, err := io.Copy(dst, io.TeeReader(src, progressBar))
	//if err != nil {
	//	return 0, err
	//}
	//
	//progressBar.Finish()

	return 0, nil
}

func main() {
	file, err := os.Open("source.txt")
	if err != nil {
		// 错误处理
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		// 错误处理
		return
	}
	fileSize := fileInfo.Size()

	destFile, err := os.Create("destination.txt")
	if err != nil {
		// 错误处理
		return
	}
	defer destFile.Close()

	progress, err := copyWithProgressBar(destFile, file, fileSize)
	if err != nil {
		// 错误处理
		return
	}

	fmt.Printf("Copy completed: %d bytes\n", progress)
}
