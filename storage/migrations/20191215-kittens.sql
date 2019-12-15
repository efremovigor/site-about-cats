create table kitten
(
	kitten_id  INTEGER PRIMARY KEY AUTOINCREMENT,
	type varchar(255) not null,
	kitten_img_id int,
	modified datetime default "0000-00-00 00:00:00"
);

create table kitten_img
(
	kitten_img_id INTEGER PRIMARY KEY AUTOINCREMENT,
	url varchar(255) not null,
	modified datetime default "0000-00-00 00:00:00"
);