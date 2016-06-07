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

## Dependencies

<pre>
make dependencies
</pre>


## Build docker image

To build a Debian based image:

<pre>
make docker
</pre>

To build an image from scratch, first copy your ca-certificates.crt file into this directory - you can find it in
/etc/ssl/certs/ in Ubuntu for example. Then:

<pre>
make smalldocker
</pre>

## Run in container

<pre>
docker run -e QUEUE_URL=https://sqs.us-east-1.amazonaws.com/930295567417/juneq -e DB_HOST=eventstoredb -e DB_NAME=esdbcopy -e DB_PASSWORD=uh-huh -e DB_PORT=5432 -e DB_USER=escopyusr --link eventstoredb:postgres  dasmith/e2pgs
</pre>

## Running it using AWS ECS

Use the task definition in e2pgs, noting you will need to customize the environment variables.

For the security to work you need to supplement the IAM role used to launch with an inline policy that grants ReceiveMessage
and DeleteMessage to the subscribed queue, with the policy attached to the  the AmazonEC2ContainerServiceforEC2Role role.

You'll need to align the placement of the launched image (used the ecs optimized image) in a VPC subnet with the
allowed CIDR/IP config for the RDS.

A VPC design for the RDS will need a VPC with two subnets - one public, and one private. The public subnet is used to
launch a bastion host that can be used to ssh into cluster instances, or can connect to the RSD for schema defs, etc. A
gateway will be needed to allow the private subnet access to dockerhub to pull the task image.
