package indexer

// PathToken tells us what a given path element means for building out the
// project + "public path" to an archived file
type PathToken int

// Path tokens an Indexer understands
const (
	Ignored PathToken = iota // folders which are "collapsed"
	Project                  // folder which defines the project name; there must be only one
	Date                     // folder describes the date files were archived in YYYY-MM-DD format
)

// Config is used to define the configuration for an indexer
type Config struct {
	DARoot           string      // Root path to the dark archive
	PathFormat       []PathToken // e.g., "project/ignore/date" would be [Project, Ignored, Date]
	InventoryPattern string      // e.g., "*/INVENTORY/*.csv" would find stuff in [anything]/INVENTORY/[anything].csv
}
