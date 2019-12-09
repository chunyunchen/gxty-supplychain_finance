#!/bin/bash

cur_dir=$(dirname BASH_SOURCE[0])
root_dir=$(dirname $(cd $cur_dir && pwd))
config_dir=$root_dir/config
couchdb_compose_file=$config_dir/docker-compose-couchdb.yaml

docker container prune -f
sleep 1

ps -ef | grep -w couchdb  | grep -vq grep
if [[ $? -ne 0 ]]; then
  docker container ls | grep -q couchdb
  if [[ $? -ne 0 ]]; then
    docker-compose -f $couchdb_compose_file up -d
  fi
fi

