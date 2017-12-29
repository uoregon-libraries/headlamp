package main

import (
	"db"
	"time"
)

func main() {
	var a = &Archiver{
		conf: getCLI(),
		dbh:  db.New(),
	}

	for {
		a.RunNextArchiveJob()
		time.Sleep(time.Minute)
	}
}
