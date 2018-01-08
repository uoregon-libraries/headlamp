package config

// PathToken tells us what a given path element means for building out the
// category + "public path" to an archived file
type PathToken int

// Path tokens an Indexer understands
const (
	Ignored  PathToken = iota // folders which are "collapsed"
	Category                  // folder which defines the category name; there must be only one
	Date                      // folder describes the date files were archived in YYYY-MM-DD format
)
