package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"path"
)

type sourcesFlags []string

func (i *sourcesFlags) String() string {
	return ""
}

func (i *sourcesFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var sources sourcesFlags

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}

	return false
}

func main() {
	flag.Var(&sources, "source", "squashfs source to serve")
	port := flag.Int("port", 0, "port to listen to")
	uRLPrefix := flag.String("url-prefix", "", "Prefix for HTTP URLs")
	flag.Parse()

	if len(sources) == 0 || *port == 0 {
		log.Fatal("At least one source and a port are required.")
	}

	var matches []string

	for _, source := range sources {
		reader := imageReader{}

		log.Print("Processing source " + source + " ...")

		err := reader.open(source)
		if err != nil {
			log.Fatal(err)
		}

		match := path.Clean(path.Join("/", *uRLPrefix, reader.hashSum))

		if contains(matches, match) {
			log.Print("Ignoring " + source + " which has a hash that was already seen")
		} else {
			log.Print("Serving " + source + " (type " + reader.fileType + ") on " + match)
			http.HandleFunc(match, reader.httpHandler)
			matches = append(matches, match)
		}
	}

	log.Printf("Listening on port %d ...", *port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	if err != nil {
		log.Fatal(err)
	}
}
