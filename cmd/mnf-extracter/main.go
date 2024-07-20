package main

import (
	"github.com/eso-tools/eso-tools/cmd/mnf-extracter/debugMnf"
	"github.com/eso-tools/eso-tools/cmd/mnf-extracter/dumpIndex"
	"github.com/eso-tools/eso-tools/cmd/mnf-extracter/dumpMnf"
	"github.com/eso-tools/eso-tools/cmd/mnf-extracter/extractAll"
	"github.com/eso-tools/eso-tools/cmd/mnf-extracter/extractFile"
	"github.com/eso-tools/eso-tools/cmd/mnf-extracter/parseLng"
	"github.com/eso-tools/eso-tools/cmd/mnf-extracter/testZosft"
	"github.com/eso-tools/eso-tools/cmd/mnf-extracter/writeLng"
	"github.com/new-world-tools/go-oodle"
	go_app "github.com/zelenin/go-app"
	"log"
)

func main() {
	if !oodle.IsDllExist() {
		err := oodle.Download()
		if err != nil {
			log.Fatalf("no oo2core library")
		}
	}

	app := go_app.NewApp()

	app.AddHandler(go_app.CommandChecker("testZosft"), testZosft.Command)
	app.AddHandler(go_app.CommandChecker("dumpMnf"), dumpMnf.Command)
	app.AddHandler(go_app.CommandChecker("dumpIndex"), dumpIndex.Command)
	app.AddHandler(go_app.CommandChecker("debugMnf"), debugMnf.Command)
	app.AddHandler(go_app.CommandChecker("extractAll"), extractAll.Command)
	app.AddHandler(go_app.CommandChecker("extractFile"), extractFile.Command)
	app.AddHandler(go_app.CommandChecker("parseLng"), parseLng.Command)
	app.AddHandler(go_app.CommandChecker("writeLng"), writeLng.Command)

	err := app.Run()
	if err != nil {
		log.Fatalf("app.Run: %s", err)
	}
}
