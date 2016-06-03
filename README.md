## Events2PGStore

This project reads events from a subscribed queue and writes the event data to PG Event Store

create database esdbcopy;
create user escopyusr with password 'uh-huh';

create table events (
    aggregate_id varchar(60)not null,
    version integer not null,
    typecode varchar(30) not null,
    payload bytea,
    primary key(aggregate_id,version)
);

grant select, insert on events to escopyusr;
