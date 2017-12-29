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
		a.CleanOldArchives()
		time.Sleep(time.Minute * 5)
	}
}
