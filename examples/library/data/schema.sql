drop table if exists authors;

create table authors (
  id INTEGER PRIMARY KEY,
  name TEXT,
  university_id INTEGER
);

drop table if exists books;

create table books (
  id INTEGER PRIMARY KEY,
  title TEXT,
  author INTEGER NOT NULL,
  series INTEGER,
  page_count INTEGER NOT NULL
);