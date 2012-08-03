create table if not exists account (
	accountName text not null unique primary key,
	passwordHash text not null,
	objectId blob not null
);

create table if not exists object (
	id blob not null,
	name text not null,
	location blob
);

insert or replace into object (id, name, location) values (
	x'00000000000000000000000000000000',
	'Room Zero',
	null
);
