drop table if exists functions;
drop table if exists configs;
drop table if exists folders;
drop table if exists http;
drop table if exists methods;
drop table if exists metrics;

create table functions (
  id integer PRIMARY KEY autoincrement,
  name varchar not null unique
);

create table configs (
  id integer PRIMARY KEY autoincrement,
  timeout integer not null,
  config blob not null,
  foreign key (id) references functions (id)
);

create table folders (
  id integer primary key autoincrement,
  label varchar not null unique
);

-- simulate enums
create table http (
  id integer primary key autoincrement,
  method varchar unique not null
);

insert into http (method) values 
  ("GET"), ("POST"), ("PUT"), ("PATCH"), ("DELETE")
;
create table methods (
  id integer PRIMARY KEY,
  method id not null,
  foreign key (id) references functions (id),
  foreign key (method) references http (id)
);

create table metrics (
  id integer primary key,
  function_id integer not null,
  called_at timestamp not null,
  duration integer not null,
  foreign key (function_id) references functions (id)
);

