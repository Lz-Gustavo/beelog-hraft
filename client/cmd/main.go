package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"beelog-hraft/client"
)

var configFilename *string

func init() {
	configFilename = flag.String("config", "client-config.toml", "Filepath to toml file")
}

func main() {
	flag.Parse()
	if *configFilename == "" {
		log.Fatalln("must set a config filepath: ./client -config '../config.toml'")
	}

	cluster, err := client.New(*configFilename)
	if err != nil {
		log.Fatalf("failed to find config: %s", err.Error())
	}

	err = cluster.Connect()
	if err != nil {
		log.Fatalf("failed to connect to cluster: %s", err.Error())
	}

	err = cluster.StartUDP()
	if err != nil {
		log.Fatalf("failed to initialize UDP socket: %s", err.Error())
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		text, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("input reader failed: %s", err.Error())
			continue
		}
		if strings.HasPrefix(text, "exit") {
			cluster.Shutdown()
			break
		}
		err = cluster.Broadcast(text + "\n")
		if err != nil {
			log.Printf("broadcast failed: %s", err.Error())
			continue
		}

		repply, _ := cluster.ReadUDP()
		fmt.Printf("Received message: %s", repply)
	}
}
