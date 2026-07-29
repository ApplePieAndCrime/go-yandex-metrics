package regular

import (
	logging "log"
	operatingsystem "os"
)

func forbidden() {
	panic("boom")           // want "использование встроенной функции panic запрещено"
	logging.Fatal("fatal")  // want "вызов log.Fatal разрешён только в функции main пакета main"
	operatingsystem.Exit(1) // want "вызов os.Exit разрешён только в функции main пакета main"
}

func notPackageCalls() {
	log := localLogger{}
	log.Fatal("allowed")
	panic := func(string) {}
	panic("allowed")
}

type localLogger struct{}

func (localLogger) Fatal(string) {}
