create table devices
(
    id         SERIAL PRIMARY KEY,
    name       varchar(250) not null,
    device_url varchar(250) not null
);

create table device_heartbeats
(
    id         SERIAL PRIMARY KEY,
    device_id  bigint                                 not null
        constraint device_heartbeats_device__fk
            references devices
            on delete restrict,
    created_at timestamp with time zone default now() not null
);

create index device_heartbeats__created_at__index
    on device_heartbeats (created_at);

create index device_heartbeats__device_id__idx
    on device_heartbeats (device_id);

create table detections
(
    id         SERIAL PRIMARY KEY,
    device_id  bigint       not null
        constraint detections_device__fk
            references devices
            on delete restrict,
    created_at timestamp with time zone default now() not null,
    label      varchar(250) NOT NULL,
    confidence float        NOT NULL DEFAULT 0.0
);

create index detections__created_at__index
    on detections (created_at);

create index detections__device_id__idx
    on detections (device_id);
