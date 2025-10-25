create table devices
(
    id         bigint       not null
        constraint device__pk
            primary key,
    name       varchar(250) not null,
    device_url varchar(250) not null
);

create table device_heartbeats
(
    id         bigint                                 not null
        constraint device_heartbeats_pk
            primary key,
    device_id  bigint                                 not null
        constraint device_heartbeats_device__fk
            references devices
            on delete restrict,
    created_at timestamp with time zone default now() not null
);

create index device_heartbeats__created_at__index
    on device_heartbeats (created_at);


create table detections
(
    id         bigint       not null
        constraint detections_pk
            primary key,
    device_id  bigint       not null
        constraint detections_device__fk
            references devices
            on delete restrict,
    created_at timestamptz           DEFAULT now() NOT NULL,
    label      varchar(250) NOT NULL,
    confidence float        NOT NULL DEFAULT 0.0
);

create index detections__created_at__index
    on device_heartbeats (created_at);