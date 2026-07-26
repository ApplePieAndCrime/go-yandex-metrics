package regular

import (
	logging "log"
	operatingsystem "os"
)

func forbidden() {
	panic("boom")
	logging.Fatal("fatal")
	operatingsystem.Exit(1)
}

func notPackageCalls() {
	log := localLogger{}
	log.Fatal("allowed")
	panic := func(string) {}
	panic("allowed")
}

type localLogger struct{}

func (localLogger) Fatal(string) {}
