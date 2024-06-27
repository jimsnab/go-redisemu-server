package main

import (
	"context"
	"os"

	"github.com/jimsnab/go-cmdline"
	"github.com/jimsnab/go-lane"
	"github.com/jimsnab/go-redisemu"
	"golang.org/x/term"
)

var keyboardInput = make(chan byte, 1000)

func main() {
	cl := cmdline.NewCommandLine()

	cl.RegisterCommand(
		mainHandler,
		"~ [<string-file>]?Runs an emulated Redis server. Specify <file> to persist data to disk.",
		"[--trace]?Enable trace logging",
		"[--port <int-port>]?Specify the TCP port to listen on. The default is 6379.",
		"[--endpoint <string-interface>]?Specify the network interface to listen on. The default is all network interfaces.",
	)

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}

	args := os.Args[1:] // exclude executable name in os.Args[0]
	err = cl.Process(args)
	term.Restore(int(os.Stdin.Fd()), oldState)
	if err != nil {
		cl.Help(err, "go-redisemu-server", args)
	}
}

func mainHandler(args cmdline.Values) error {
	go stdinReader() // can't stop this go routine thanks to go design

	l := lane.NewLogLaneWithCR(context.Background())

	isTrace := args["--trace"].(bool)
	if !isTrace {
		l.SetLogLevel(lane.LogLevelInfo)
	}

	port := args["port"].(int)
	if port == 0 {
		port = 6379
	}
	iface := args["interface"].(string)
	basePath := args["file"].(string)

	stopSignal := make(chan struct{}, 1000)
	go stdinReaderAdapter(l, stopSignal)

	redisServer, err := redisemu.NewEmulator(
		l,
		port,
		iface,
		basePath,
		stopSignal,
	)
	if err != nil {
		panic(err)
	}

	redisServer.Start()
	redisServer.WaitForTermination()
	return nil
}

func stdinReader() {
	for {
		b := []byte{0}
		n, err := os.Stdin.Read(b)
		if err != nil {
			return
		}

		if n == 1 {
			keyboardInput <- b[0]
		}
	}
}

func stdinReaderAdapter(l lane.Lane, signal chan struct{}) {
	for {
		select {
		case <-l.Done():
			return
		case <-keyboardInput:
			// trigger close on whatever was typed
			signal <- struct{}{}
		}
	}
}
