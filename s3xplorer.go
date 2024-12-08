package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sgaunet/s3xplorer/pkg/app"
	"github.com/sgaunet/s3xplorer/pkg/config"
)

func main() {
	var err error
	var fileName string
	var cfg config.Config
	flag.StringVar(&fileName, "f", "", "Configuration file")
	flag.Parse()

	if fileName == "" {
		fmt.Println("Configuration file not provided. Exit 1")
		fmt.Printf("\nUsage:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if cfg, err = config.ReadYamlCnxFile(fileName); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Handle SIGTERM/SIGINT
	ctx, cancelFunc := context.WithCancel(context.Background())
	SetupCloseHandler(ctx, cancelFunc)

	s, err := app.NewApp(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	// os.Exit(0)
	<-ctx.Done()
	fmt.Println("INFO: stop server http and close DB connection")
	s.StopServer()
}

func SetupCloseHandler(ctx context.Context, cancelFunc context.CancelFunc) {
	c := make(chan os.Signal, 5)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-c
		fmt.Println("INFO: Signal received:", s)
		cancelFunc()
	}()
}
