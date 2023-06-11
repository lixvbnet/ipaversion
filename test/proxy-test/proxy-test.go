package main

import (
	"github.com/lqqyt2423/go-mitmproxy/proxy"
	"log"
)

func main() {
	opts := &proxy.Options{
		Addr:              ":9080",
		StreamLargeBodies: 1024 * 1024 * 5,
	}

	p, err := proxy.NewProxy(opts)
	if err != nil {
		log.Fatal(err)
	}

	// Add on
	p.AddAddon(&proxy.LogAddon{})

	log.Fatal(p.Start())
}
