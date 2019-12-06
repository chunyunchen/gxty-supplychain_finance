#!/bin/bash

cur_dir=$(dirname BASH_SOURCE[0])
root_dir=$(dirname $(cd $cur_dir && pwd))

echo "ROOT DIR: $root_dir"

function clean_artifacts()
{
    rm -rf $root_dir/config/channel-artifacts/*
    rm -rf $root_dir/config/crypto-config
    rm -rf $root_dir/data/*
    return 0
}

function gen_artifacts()
{
    mkdir -p $root_dir/data/peer $root_dir/data/orderer
    return 0
}

clean_artifacts
