## Events2PGStore

This project reads events from a subscribed queue and writes the event data to PG Event Store

## Schema

In a nutshell the events table from pgeventstore is used. A good set up is to use a separate schema
and user to make sure you're not replicating your source into your source...

<pre>
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
</pre>

## Build docker image

<pre>
docker build -t dasmith/e2pgs .
</pre>

## Run in container

<pre>
docker run -e QUEUE_URL=https://sqs.us-east-1.amazonaws.com/930295567417/juneq -e DB_HOST=eventstoredb -e DB_NAME=esdbcopy -e DB_PASSWORD=uh-huh -e DB_PORT=5432 -e DB_USER=escopyusr --link eventstoredb:postgres  dasmith/e2pgs
</pre>
