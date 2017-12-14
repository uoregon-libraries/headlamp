-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE projects (
  id integer not null primary key,
  name text not null
);

CREATE INDEX projects_name ON projects (name);

CREATE TABLE inventories (
  id integer not null primary key,
  path text not null
);

-- Folders are a convenience concept that don't map to the filesystem directly.
-- A file is always within a project's structure somewhere, has an archive-date
-- folder, and can have any number of "ignored" / collapsed folders in the
-- tree.  The "folders" table just indexes all the non-collapsed folders a file
-- has in its path so we can present users with a fake filesystem that's
-- hopefully easier to handle.
CREATE TABLE folders (
  id integer not null primary key,
  project_id integer not null,
  folder_id integer not null,
  depth integer not null,

  -- This is the name of the folder for easier display
  name text not null,

  -- This is the "public" path we expose to users for searching; it collapses
  -- things like project directory, volume directories, date dir, etc.
  public_path text not null
);

CREATE INDEX folders_public_path ON folders (public_path);
CREATE INDEX folders_folder_id ON folders (folder_id);
CREATE INDEX folders_depth ON folders (depth);
CREATE UNIQUE INDEX folders_unique ON folders (project_id, public_path);

CREATE TABLE files (
  id integer not null primary key,
  project_id integer not null,
  inventory_id integer not null,
  folder_id integer not null,
  depth integer not null,
  archive_date text not null,

  -- Data integrity info
  checksum text not null,
  filesize integer not null,

  -- This is the name of the file for easier display
  name text not null,

  -- This is the full path (relative to the dark archive root)
  full_path text not null,

  -- This is the "public" path we expose to users for searching; it collapses
  -- things like project directory, volume directories, archive date
  -- directories, etc.
  public_path text not null
);

CREATE INDEX files_public_path ON files (public_path);
CREATE INDEX files_project_id ON files (project_id);
CREATE INDEX files_folder_id ON files (folder_id);
CREATE INDEX files_inventory_id ON files (inventory_id);
CREATE INDEX files_depth ON files (depth);
CREATE UNIQUE INDEX files_unique ON files (project_id, archive_date, public_path);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE files;
DROP TABLE folders;
DROP TABLE inventories;
DROP TABLE projects;
