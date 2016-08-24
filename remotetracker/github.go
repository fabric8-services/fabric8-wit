package remotetracker

import "fmt"

// Github represents github remote issue tracker
type Github struct {
	data []byte
}

func (g *Github) Fetch(url string, query string) error {
	fmt.Println("Hello Github")
	return nil
}

func (g *Github) Import() error {
	fmt.Println("Import Github")
	return nil
}
