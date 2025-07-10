package main

import (
	"flag"
	"github.com/shaco-go/gkit-layout/bootstrap"
)

func main() {
	path := flag.String("c", "configs/development.yaml", "config file path")
	flag.Parse()
	bootstrap.Init(*path)
}
