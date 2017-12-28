-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE zip_jobs (
  id integer not null primary key,
  created_at datetime not null,
  files text not null,
  notification_emails text not null
);
CREATE INDEX zip_jobs_created_at ON zip_jobs (created_at);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE zip_jobs;
