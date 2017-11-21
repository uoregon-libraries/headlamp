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

CREATE TABLE folders (
  id integer not null primary key,
  project_id integer not null,
  folder_id integer not null,

  -- This is the name of the folder for easier display
  name text not null,

  -- This is the "public" path we expose to users for searching; it collapses
  -- things like project directory, volume directories, date dir, etc.
  path text not null
);

CREATE INDEX folders_path ON folders (path);
CREATE INDEX folders_folder_id ON folders (folder_id);
CREATE UNIQUE INDEX folders_unique ON folders (project_id, path);

CREATE TABLE files (
  id integer not null primary key,
  project_id integer not null,
  inventory_id integer not null,
  folder_id integer not null,
  archive_date datetime not null,

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
CREATE UNIQUE INDEX files_unique ON files (project_id, archive_date, public_path);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE files;
DROP TABLE folders;
DROP TABLE inventories;
DROP TABLE projects;
