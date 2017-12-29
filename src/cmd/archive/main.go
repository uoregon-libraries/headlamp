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
		a.RunPendingArchiveJobs()
		time.Sleep(time.Minute * 5)
	}
}
