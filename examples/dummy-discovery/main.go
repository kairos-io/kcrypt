package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jaypipes/ghw/pkg/block"
	"github.com/keirros-io/kcrypt/pkg/bus"

	"github.com/mudler/go-pluggable"
)

func main() {
	if len(os.Args) >= 2 && bus.IsEventDefined(os.Args[1]) {
		checkErr(start())
	}
}

func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}

func start() error {
	factory := pluggable.NewPluginFactory()

	// Input: bus.EventInstallPayload
	// Expected output: map[string]string{}
	factory.Add(bus.EventDiscoveryPassword, func(e *pluggable.Event) pluggable.EventResponse {
		b := &block.Partition{}
		var errString string
		err := json.Unmarshal([]byte(e.Data), b)
		if err != nil {
			errString = err.Error()
		}

		return pluggable.EventResponse{
			Data:  "hardcoded password",
			Error: errString,
		}
	})

	return factory.Run(pluggable.EventType(os.Args[1]), os.Stdin, os.Stdout)
}
