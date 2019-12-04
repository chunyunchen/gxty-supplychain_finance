#!/bin/bash

root_dir=$(dirname $(dirname ${BASH_SOURCE[0]}))
echo $root_dir
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
gen_artifacts
