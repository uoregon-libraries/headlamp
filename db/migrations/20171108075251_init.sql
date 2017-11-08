-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE projects (
  id integer not null primary key,
  name text not null
);

CREATE INDEX projects_name ON projects (name);

CREATE TABLE inventories (
  id integer not null primary key,
  project_id integer not null,
  filename text not null
);

CREATE INDEX inventories_project_id ON inventories (project_id);

CREATE TABLE files (
  id integer not null primary key,
  project_id integer not null,
  inventory_id integer not null,
  archive_date date not null,

  -- Relative path to the file, relative to project/date/
  checksum text not null,
  filesize integer not null,
  path text not null
);

CREATE INDEX files_path ON files (path);
CREATE INDEX files_project_id ON files (project_id);
CREATE INDEX files_inventory_id ON files (inventory_id);
CREATE UNIQUE INDEX files_unique ON files (project_id, archive_date, path);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE files;
DROP TABLE inventories;
DROP TABLE projects;
