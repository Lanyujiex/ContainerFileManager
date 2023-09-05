package filemanager

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	ctrl "sigs.k8s.io/controller-runtime"
)

var config *rest.Config

var clientset *kubernetes.Clientset

type FileManager struct {
}

func init() {
	var err error

	config = ctrl.GetConfigOrDie()
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}
}

// 不好使
func (f *FileManager) UploadHandlerChunk(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "测试上传", r.Method)
	err := r.ParseMultipartForm(10 << 20) // limit your max input length!
	if err != nil {
		fmt.Println("Could not parse multipart form: ", err)
		return
	}

	fileStream, err := createIOStream(r)
	if err != nil {
		fmt.Fprintln(w, err)
	}
	namespace := r.URL.Query().Get("namespace")
	podName := r.URL.Query().Get("podName")
	containerName := r.URL.Query().Get("containerName")
	fileName := r.URL.Query().Get("fileName")
	filePath := r.URL.Query().Get("filePath")
	final := r.URL.Query().Get("final") == "final"
	first := r.URL.Query().Get("first") == "first"
	ChunKPool(containerName, podName, namespace, filePath, fileName, "aaa", &fileStream, final, first)

}

func (f *FileManager) UploadHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "测试上传", r.Method)
	err := r.ParseMultipartForm(10 << 20) // limit your max input length!
	if err != nil {
		fmt.Println("Could not parse multipart form: ", err)
		return
	}

	fileStream, err := createIOStream(r)
	//fileStream, err := createTarStream(r)
	if err != nil {
		fmt.Println(err)
		return
	}
	namespace := r.URL.Query().Get("namespace")
	podName := r.URL.Query().Get("podName")
	containerName := r.URL.Query().Get("containerName")
	fileName := r.URL.Query().Get("fileName")
	filePath := r.URL.Query().Get("filePath")
	opId := r.URL.Query().Get("opId")
	start := r.URL.Query().Get("start") == "true"

	var cmd string
	cmd = fmt.Sprintf("cat >> '%s'", fmt.Sprintf("%s/%s", filePath, fileName))
	if start {
		cmd = fmt.Sprintf("cat > '%s'", fmt.Sprintf("%s/%s", filePath, fileName))
	}
	//cmd = strings.Join([]string{"tar", "-xmf", "-", "-C", "/","cat "}, " ")

	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			Container: containerName,
			Command:   []string{"sh", "-c", cmd},
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute) // 增加超时时间为 5 分钟
	//defer cancel()
	cr := &CountingReader{Reader: fileStream}
	finishChan := make(chan struct{})
	key := getKey(containerName, podName, namespace, filePath, fileName, opId)

	go ProgressBar(cr, finishChan, key)

	err = executor.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdin:  cr,
		Stdout: w,
		Stderr: os.Stderr,
		Tty:    false,
	})
	finishChan <- struct{}{}
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Fprintf(w, "Upload successfully completed")
}

func (f *FileManager) DownloadHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("测试下载")
	var cmd string

	namespace := r.URL.Query().Get("namespace")
	podName := r.URL.Query().Get("podName")
	containerName := r.URL.Query().Get("containerName")
	fileName := r.URL.Query().Get("fileName")
	filePath := r.URL.Query().Get("filePath")
	cmd = fmt.Sprintf("cat '%s'", fmt.Sprintf("%s/%s", filePath, fileName))

	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   []string{"sh", "-c", cmd},
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = executor.Stream(remotecommand.StreamOptions{
		Stdout: w,
		Stderr: os.Stderr,
		Tty:    false,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (f *FileManager) ListDir(w http.ResponseWriter, r *http.Request) {
	fmt.Println("测试查询")
	var cmd string

	namespace := r.URL.Query().Get("namespace")
	podName := r.URL.Query().Get("podName")
	containerName := r.URL.Query().Get("containerName")
	//fileName := r.URL.Query().Get("fileName")
	filePath := r.URL.Query().Get("filePath")
	//cmd = fmt.Sprintf("ls -lQ --color=never '%s'", fmt.Sprintf("%s/%s", filePath, fileName))
	cmd = fmt.Sprintf("ls -lQ --color=never '%s'", filePath)
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   []string{"sh", "-c", cmd},
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = executor.Stream(remotecommand.StreamOptions{
		Stdout: w,
		Stderr: os.Stderr,
		Tty:    false,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
