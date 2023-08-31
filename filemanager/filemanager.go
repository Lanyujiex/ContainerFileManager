package filemanager

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"log"
	"net/http"
	"os"
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

func (f *FileManager) UploadHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintln(w, "测试上传", r.Method)

	err := r.ParseMultipartForm(10 << 20) // limit your max input length!
	if err != nil {
		fmt.Println("Could not parse multipart form: ", err)
		return
	}
	//var opFile *multipart.File
	////get the *fileheaders
	//files := r.MultipartForm.File["myFile"] //grab the filenames
	//for _, fileHeader := range files {
	//	// for each fileheader, get a handle to the actual fileHeader
	//	file, err := fileHeader.Open()
	//	if err != nil {
	//		fmt.Println("Error retrieving the fileHeader ", fileHeader.Filename)
	//		fmt.Println(err)
	//		return
	//	}
	//	opFile = &file
	//	defer file.Close()
	//
	//}
	//tarStream, err := createTarStream(r)
	//if err != nil {
	//	fmt.Fprintln(w, err)
	//}
	fileStream, err := createIOStream(r)
	if err != nil {
		fmt.Fprintln(w, err)
	}
	namespace := r.URL.Query().Get("namespace")
	podName := r.URL.Query().Get("podName")
	containerName := r.URL.Query().Get("containerName")
	fileName := r.URL.Query().Get("fileName")
	filePath := r.URL.Query().Get("filePath")
	//cmd := strings.Join([]string{"tar", "-xmf", "-", "-C", "/"}, " ")
	cmd := fmt.Sprintf("cat > '%s'", fmt.Sprintf("%s/%s", filePath, fileName))
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
			//Command: []string{"sh", "-c", "ls -l /"},
		}, scheme.ParameterCodec)

	pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		return
	}
	fmt.Fprintf(w, fmt.Sprintf("%s\n", pod.Name))
	executor, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 创建文件
	//err = createFile("outfile", &fileStream)
	if err != nil {
		return
	}

	//ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute) // 增加超时时间为 5 分钟
	//defer cancel()

	err = executor.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		//Stdin:  file,
		Stdin:  fileStream,
		Stdout: w,
		Stderr: os.Stderr,
		Tty:    false,
	})
	if err != nil {
		fmt.Fprintln(w, err.Error())
		return
	}

	fmt.Fprintf(w, "Upload successfully completed")
}

func (f *FileManager) DownloadHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "测试下载")

	queryParams := r.URL.Query()
	filename, present := queryParams["filename"]
	if !present || len(filename) != 1 {
		http.Error(w, "filename query param missing or multiple provided", http.StatusBadRequest)
		return
	}

	podName := "podName"     // fill this
	containerName := "nginx" // and this, according to your situation
	namespace := "default"   // and this too

	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   []string{"cat", "/root/" + filename[0]},
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
