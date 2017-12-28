Headlamp
===

Shining light into your dark archive since 2017.  Or 2018.  Or whenever this
ends up being usable.

This project is very UO-library-centric, made open-source so others can see how
we made a "digital preservation discovery system" on the cheap more than to be
a general-purpose tool others could just grab and use.

That said, if you do stuff the way we do stuff, you could theoretically use
this as-is....

Quick Usage
---

### Install Go

Install [Go](https://golang.org/dl/), preferably 1.9 or later.  Headlamp may
work on older versions, but I haven't tested it.

Note that you can install Go without root access just by unpacking one of the
binary tarballs and exporting some environment variables, such as:

    export GOROOT=$HOME/go
    export GOPATH=$HOME/projects/go
    export PATH=$PATH:$GOROOT/bin:$GOPATH/bin

### Prerequisites

    go get github.com/constabulary/gb/...
    go get bitbucket.org/liamstask/goose/cmd/goose
    git clone https://github.com/uoregon-libraries/headlamp.git
    cd headlamp
    goose up
    make

### Prepare Settings

Copy `settings_example` to `settings` and modify it as needed.  The comments in
that file should clearly describe what each setting means, but more explanation
for some of the indexer's settings can be found below.

### Index your data

Run the indexer; this takes a few minutes for us on the first run, scanning
about four million file entries.  The indexer runs forever, scanning for new
files it hasn't yet indexed.

    ./bin/index settings

### Start the web server

    ./bin/headlamp settings

Inventory Files
---

The key to Headlamp is the inventory files generated when batches are
uploaded to the dark archive.  These files are how Headlamp knows what
exists; scanning the entire dark archive would be a significantly slower
process.

We have a script somewhere (which should be migrated here one day) which
transfers a batch to the dark archive after generating a pseudo-CSV file
containing the following data:

- Checksum (SHA256 in our case)
- File size in bytes
- Filename

Since some filenames have commas in them, we can't call this a true CSV file,
but it's easy enough to just split on the first two commas and understand that
everything after that second comma is the filename.

The filename itself is a relative path from the *parent* of the directory which
contained the inventory file.  So in our world, we might have
`/path/to/dark-archive/foo/projectname/INVENTORY/Archive-2017-12-08.csv`.  The
files described therein would be found relative to
`/path/to/dark-archive/foo/projectname`.

As a special case, Headlamp will **not process or look at or even offer a
friendly wave to** any files called `manifest.csv`!  That file, for UO,
contains a comprehensive list of all other inventories as an easier way to do
things like data-rot detection.

Also note that the location of inventory files *must be consistent*.  When
configuring the indexer, you must specify a pattern for finding these files
relative to the dark archive root (`INVENTORY_FILE_GLOB`).  For instance, we
might say our pattern above is `*/*/INVENTORY/*.csv` (assuming the dark archive
root was `/path/to/dark-archive/`).  Though multiple indexers could be run to
grab different patterns, it could become confusing to manage them.

Directory Format
---

For our dark archive design, we decided that we didn't want to be trying to
dedupe files (the filesystem can handle that for us) or worry about accidental
overwriting of a filename.  In fact, the dark archive is effectively a
write-only system for us unless we have a catastrophic failure.

Therefore, to avoid these complexities, we have the archive date as one of the
path elements.  e.g., we might have a filename of
`2017-12-07/FILES/385_foo_bar.tiff`.  The original path was just
`FILES/385_foo_bar.tiff`, but adding the date makes the "send stuff to dark
archive" scripting far simpler.

We additionally expect every file to live under a project of some kind.  For
telling the indexer how to find files, there are a few key points here:

- All files' "real" paths are the full path to the file *minus* the dark archive root
- All files must have a project name in their full path somewhere
- All files must have a path element in the format of YYYY-MM-DD denoting the archive date

When presenting data to the end user, we felt that we needed to prioritize the
project name, making it act as a top-level element.  We also ignore the archive
date portion of the path so users aren't having to guess at when a given file
was archived.

Running the indexer requires telling it your path strategy using special
keywords "project", "date", and "ignore". The order doesn't matter, so long as
you have exactly one project and date.  You can have any number of ignored
paths, or none at all.

To explain through example, given the following:

- The dark archive root is at `/path/to/archive`
- The inventory files are at `*/INVENTORY/*.csv`; e.g., from the dark archive
  root, you might have `foo/INVENTORY/archive.csv`
- A file defined in that inventory is `2017-12-08/baz/srs/FILES/blah.tiff`
- The path format (`ARCHIVE_PATH_FORMAT` in your settings file) is `ignore/date/ignore/project`

Then the following are true:

- The inventory file's full path is `/path/to/archive/foo/INVENTORY/archive.csv`
- The file's full path is `/path/to/archive/foo/2017-12-08/bar/srs/FILES/blah.tiff`
- The file's archive date is December 8th, 2017
- The file lives under the project "srs"
- The path elements "foo" and "bar" are ignored
- The user, via the web discovery tool, would find this file under the "srs" project at `FILES/blah.tiff`
