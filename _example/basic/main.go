package main

import (
	"os/signal"
	"syscall"

	"github.com/logbn/zongzi"
)

func main() {
	agent, err := zongzi.NewAgent("zzexample01", []string{
		"node-1:10801",
		"node-2:10801",
		"node-3:10801",
	})
	if err != nil {
		panic(err)
	}
	if err = agent.Start(); err != nil {
		panic(err)
	}

	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	signal.Notify(stop, syscall.SIGTERM)
	<-stop
	agent.Stop()
}
