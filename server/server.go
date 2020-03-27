package main

import (
	"flag"
	"fmt"
	"net/http"
)

func main() {

	var listen string
	flag.StringVar(&listen, "listen", ":80", "listening address")
	flag.Parse()

	fmt.Println("Listening on", listen)

	simpleServer(listen)
}

func simpleServer(listen string) {

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "OK")
	})

	err := http.ListenAndServe(listen, nil)
	if err != nil {
		panic(fmt.Sprintf("Error in starting listener: %s\n", err))
	}
}
