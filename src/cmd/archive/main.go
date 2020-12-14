package main

import (
	"time"

	"github.com/uoregon-libraries/headlamp/src/db"
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
