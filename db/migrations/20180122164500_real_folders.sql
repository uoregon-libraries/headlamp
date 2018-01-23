-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- Real folders are the actual folders on the filesystem which map to a
-- collapsed / public folder
CREATE TABLE real_folders (
  id integer not null primary key,
  folder_id integer not null,
  full_path text not null
);

CREATE INDEX real_folders_folder_id ON real_folders (folder_id);
CREATE UNIQUE INDEX real_folders_unique ON real_folders (folder_id, full_path);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE real_folders;
