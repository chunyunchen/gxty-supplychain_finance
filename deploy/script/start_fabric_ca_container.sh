#!/bin/bash

cur_dir=$(dirname BASH_SOURCE[0])
root_dir=$(dirname $(cd $cur_dir && pwd))
config_dir=$root_dir/config
fabric_ca_compose_file=$config_dir/docker-compose-fabric-ca.yaml

ps -ef | grep -w "fabric-ca-server"| grep -vq grep
if [[ $? -ne 0 ]]; then
  docker container prune -f
  sleep 1

  docker container ls | grep -q "ca_peer"
  if [[ $? -ne 0 ]]; then
    docker-compose -f $fabric_ca_compose_file up -d
  fi
fi

