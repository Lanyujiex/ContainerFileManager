package filemanager

import (
	"context"
	"fmt"
	"io"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

var pipWriterCaches map[string]*io.PipeWriter
var pipReaderCaches map[string]*io.PipeReader

func init() {
	pipWriterCaches = map[string]*io.PipeWriter{}
	pipReaderCaches = map[string]*io.PipeReader{}
}

func RedirectStream(pipIn *io.PipeWriter, srcStream *io.Reader, final bool) {
	if final {
		defer pipIn.Close()
	}
	io.Copy(pipIn, *srcStream)
}

func ChunKPool(containerName, podName, namespace, filePath, fileName, opUid string, chunkStream *io.Reader, final, first bool) {
	key := fmt.Sprintf("%s-%s-%s-%s-%s-%s", containerName, podName, namespace, filePath, fileName, opUid)
	pipeWriter, ok := pipWriterCaches[key]
	pipeReader, ok := pipReaderCaches[key]
	if !ok {
		pipeReader, pipeWriter = io.Pipe()
		pipWriterCaches[key] = pipeWriter
		pipReaderCaches[key] = pipeReader
	}
	go RedirectStream(pipeWriter, chunkStream, final)
	if !first {
		return
	}

	go func() {
		cmd := fmt.Sprintf("cat > '%s'", fmt.Sprintf("%s%s", filePath, fileName))
		req := clientset.CoreV1().RESTClient().Post().
			Resource("pods").
			Namespace(namespace).
			Name(podName).
			SubResource("exec").
			VersionedParams(&corev1.PodExecOptions{
				Stdin:     true,
				Stdout:    false,
				Stderr:    true,
				Container: containerName,
				Command:   []string{"sh", "-c", cmd},
			}, scheme.ParameterCodec)

		pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			fmt.Println(err)
			fmt.Println(pod)
			return
		}
		fmt.Println(pod.Name)
		executor, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
		if err != nil {
			fmt.Println(err)
			return
		}
		err = executor.StreamWithContext(context.Background(), remotecommand.StreamOptions{
			Stdin:  pipeReader,
			Stdout: nil,
			Stderr: os.Stderr,
			Tty:    false,
		})
		if err != nil {
			fmt.Println(err)
		}
	}()

}
