package main

import "db"

// seenFile just gives us a zero-length value for the FileIDs map
var seenFile struct{}

// BulkFileQueue represents a user's queued files for download
type BulkFileQueue struct {
	FileIDs map[uint64]struct{} // struct{} has no size, so this is the most efficient map of "ids I have indexed"
}

// NewBulkFileQueue initializes an empty queue
func NewBulkFileQueue() *BulkFileQueue {
	return &BulkFileQueue{FileIDs: make(map[uint64]struct{})}
}

// HasFile returns true if the queue has the given file's id
func (q *BulkFileQueue) HasFile(f *db.File) bool {
	var _, ok = q.FileIDs[f.ID]
	return ok
}

// AddFile puts the given file's id into this queue
func (q *BulkFileQueue) AddFile(f *db.File) {
	q.FileIDs[f.ID] = seenFile
}

// RemoveFile takes the given file's id out of this queue
func (q *BulkFileQueue) RemoveFile(f *db.File) {
	delete(q.FileIDs, f.ID)
}

// Files attempts to load all db.File instances from the database and return
// them.  If a queue is huge, this could of course take a very long time.
func (q *BulkFileQueue) Files() ([]*db.File, error) {
	var ids []uint64
	for k := range q.FileIDs {
		ids = append(ids, k)
	}
	return dbh.Operation().GetFilesByIDs(ids)
}