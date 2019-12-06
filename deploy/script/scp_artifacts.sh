#!/bin/bash

function printHelp() {
    echo "Usange: scp_artifacts <username> <remote host ip> <remote_dir>"
    exit 1
}

if [[ $# != 3 ]]; then
    printHelp
fi

cur_dir=$(dirname $BASH_SOURCE[0])
root_dir=$(dirname $(cd $cur_dir && pwd))

script_dir=$root_dir/script
config_dir=$root_dir/config
channel_block_file=$config_dir/channel-artifacts/sfchl.block
remote_host="$1@$2:$3"

set -e

if [[ -f $channel_block_file ]]; then
  echo "copy $channel_block_file to $remote_host/config/channel-artifacts"
  scp -r $channel_block_file $remote_host/config/channel-artifacts
else
  echo "copy $script_dir to $remote_host"
  scp -r $script_dir $remote_host

  echo "copy $config_dir to $remote_host"
  scp -r $config_dir $remote_host
fi
