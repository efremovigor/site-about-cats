create table main.kitten_task
(
	kitten_task_id  INTEGER PRIMARY KEY AUTOINCREMENT,
	status int(10) not null,
	data text,
	modified datetime default "0000-00-00 00:00:00"
);
