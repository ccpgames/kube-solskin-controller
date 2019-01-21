package suppressor

import (
	"os"
)

// Start ...
func Start(stopper <-chan os.Signal) {
	<-stopper
}
