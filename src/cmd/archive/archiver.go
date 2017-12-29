package main

import (
	"archive/tar"
	"config"
	"db"
	"fmt"
	"io"
	"net/smtp"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/logger"
)

// Archiver holds the database handle and config to simplify processing
type Archiver struct {
	conf *config.Config
	dbh  *db.Database
}

// RunPendingArchiveJobs grabs the longest-waiting job and processes it
func (a *Archiver) RunPendingArchiveJobs() {
	logger.Debugf("Scanning for pending archive jobs")
	var pending = true
	for pending {
		pending = false
		var err = a.dbh.Operation().ProcessArchiveJob(func(j *db.ArchiveJob) bool {
			pending = true
			return a.processArchiveJob(j)
		})

		if err != nil {
			logger.Errorf("Unable to get next job: %s", err)
			return
		}
	}
}

// CleanOldArchives looks for old archive files and removes them
func (a *Archiver) CleanOldArchives() {
	logger.Debugf("Scanning for old archives to remove")

	var oldFiles, err = fileutil.FindIf(a.conf.ArchiveOutputLocation, func(i os.FileInfo) bool {
		if !i.Mode().IsRegular() {
			return false
		}

		var n = i.Name()
		if !strings.HasSuffix(n, ".tar") {
			return false
		}

		var hp = strings.HasPrefix
		if !hp(n, ".wip-") && !hp(n, "archive-") {
			return false
		}

		var archiveLifetime = time.Hour * 24 * time.Duration(a.conf.ArchiveLifetimeDays)
		var timeSince = time.Since(i.ModTime())
		if timeSince < archiveLifetime {
			logger.Debugf("Skipping %q: too recently modified", n)
			return false
		}

		return true
	})

	if err != nil {
		logger.Errorf("Unable to find old archives to delete: %s", err)
		return
	}

	for _, f := range oldFiles {
		logger.Infof("Removing %q", f)
		err = os.Remove(f)
		if err != nil {
			logger.Errorf("Unable to delete %q: %s", f, err)
		}
	}
}

func (a *Archiver) processArchiveJob(j *db.ArchiveJob) bool {
	logger.Infof("Processing archive job %d", j.ID)

	var tempFile, err = fileutil.TempFile(a.conf.ArchiveOutputLocation, ".wip-", ".tar")
	var tempName = tempFile.Name()
	if err != nil {
		logger.Errorf("Unable to create temp archive: %s", err)
		return false
	}

	// Most failures are before the rename, so this helps reduce chances of
	// leaving orphaned files around
	defer os.Remove(tempName)

	var tw = tar.NewWriter(tempFile)

	logger.Debugf("Adding files to archive")
	for _, fname := range j.FileList() {
		var p = filepath.Join(a.conf.DARoot, fname)
		var fn = strings.Replace(fname, string(os.PathSeparator), "__", -1)
		err = addFileToTar(tw, p, fn)
		if err != nil {
			logger.Errorf("Unable to add %q to archive: %s", fname, err)
			return false
		}
	}

	logger.Debugf("Closing archive")
	err = tw.Close()
	if err != nil {
		logger.Errorf("Error closing tar stream %q: %s", tempName, err)
		return false
	}

	logger.Debugf("Closing tempfile")
	err = tempFile.Close()
	if err != nil {
		logger.Errorf("Error closing %q: %s", tempName, err)
		return false
	}

	logger.Debugf("Generating new unique filename")
	var newName string
	newName, err = fileutil.TempNamedFile(a.conf.ArchiveOutputLocation, "archive-", ".tar")
	if err != nil {
		logger.Errorf("Unable to create second temp archive: %s", err)
		return false
	}
	os.Remove(newName)

	logger.Debugf("Notifying user(s) via email")
	var to = j.Emails()
	var archiveDownloadURL, _ = url.Parse(a.conf.WebPath)
	archiveDownloadURL.Path = path.Join(archiveDownloadURL.Path, "archives", filepath.Base(newName))
	err = a.notify(to, archiveDownloadURL.String())
	if err != nil {
		logger.Criticalf("Unable to notify %q of archive %q being ready: %s", to, archiveDownloadURL, err)
		return false
	}

	logger.Debugf("Renaming file (via os.Link)")
	err = os.Link(tempName, newName)
	if err != nil {
		logger.Errorf("Error linking %q to %q: %s", tempName, newName, err)
		return false
	}

	logger.Infof("Job %d completed successfully", j.ID)
	return true
}

func addFileToTar(tw *tar.Writer, filePath, flatname string) error {
	var srcFile, err = os.Open(filePath)
	if err != nil {
		return fmt.Errorf("os.Open(%q): %s", filePath, err)
	}

	var info os.FileInfo
	info, err = srcFile.Stat()
	if err != nil {
		return fmt.Errorf("unable to stat %q: %s", filePath, err)
	}

	var header = &tar.Header{Name: flatname, Mode: 0600, Size: info.Size()}
	err = tw.WriteHeader(header)
	if err != nil {
		return fmt.Errorf("writing header for %q: %s", flatname, err)
	}

	_, err = io.Copy(tw, srcFile)
	if err != nil {
		return fmt.Errorf("%q io.Copy(): %s", flatname, err)
	}

	return nil
}

func (a *Archiver) notify(to []string, fileURL string) error {
	var auth = smtp.PlainAuth("", a.conf.SMTPUser, a.conf.SMTPPass, a.conf.SMTPHost)
	var msg = fmt.Sprintf("Subject: Your archive is ready\r\n\r\nDownload your Headlamp archive at %s\r\n", fileURL)
	var server = fmt.Sprintf("%s:%d", a.conf.SMTPHost, a.conf.SMTPPort)
	return smtp.SendMail(server, auth, a.conf.SMTPUser, to, []byte(msg))
}
