package notifier

import (
	"k8s.io/client-go/kubernetes"
	"os"
)

// Start ...
func Start(client kubernetes.Interface, stopper <-chan os.Signal) {
	<-stopper
}
