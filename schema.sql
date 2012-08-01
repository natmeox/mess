create table if not exists account (
	accountName varchar(60) not null unique primary key,
	passwordHash varchar(30) not null,
	objectId char(16) not null
);

create table if not exists object (
	uuid integer
);
