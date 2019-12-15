create table main.kitten
(
	kitten_id  INTEGER PRIMARY KEY AUTOINCREMENT UNSIGNED,
	name varchar(255) not null,
	description text,
	modified datetime default "0000-00-00 00:00:00"
);

create table main.kitten_img
(
	kitten_img_id INTEGER PRIMARY KEY AUTOINCREMENT UNSIGNED,
	kitten_id int UNSIGNED not null,
	url varchar(255) not null,
	modified datetime default "0000-00-00 00:00:00"
);