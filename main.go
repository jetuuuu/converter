package main

import (
	"flag"
	"log"

	"github.com/jetuuuu/converter/config"
	"github.com/jetuuuu/converter/rest"
)

func main() {
	consulAddrPtr := flag.String("addr", "", "consul addres must be ip:port")
	consulPrefixPtr := flag.String("pref", "test", "consul prefix")
	flag.Parse()

	if consulAddrPtr == nil || *consulAddrPtr == "" {
		log.Fatal("Consul addres must be not empty")
		return
	}

	reader, _ := config.NewConfigReader(*consulAddrPtr, *consulPrefixPtr)
	s := rest.New(reader)
	s.Run()
}
