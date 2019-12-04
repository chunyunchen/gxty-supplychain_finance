#!/bin/bash

script_dir=$(dirname $BASH_SOURCE[0])
root_dir=$(dirname $(cd $script_dir && pwd))

host_name=$1
if [[ -z $host_name ]]; then
    echo "Start orderer service failed. Please specify orderer hostname [orderer|orderer2|orderer3] which is defined in crypto-config.yaml"
    exit 1
    echo
fi

orderer=(orderer.gtbcsf.com)
orderer2=(orderer2.gtbcsf.com)
orderer3=(orderer3.gtbcsf.com)

orderer_address_dns=$(eval echo '$'"{$host_name[0]}")

export FABRIC_LOGGING_SPEC=DEBUG
export ORDERER_GENERAL_TLS_ENABLED=true
export ORDERER_GENERAL_TLS_PRIVATEKEY=$rootDir/config/crypto-config/ordererOrganizations/gtbcsf.com/orderers/$orderer_address_dns/tls/server.key
export ORDERER_GENERAL_TLS_CERTIFICATE=$rootDir/config/crypto-config/ordererOrganizations/gtbcsf.com/orderers/$orderer_address_dns/tls/server.crt
export ORDERER_GENERAL_TLS_ROOTCAS=$rootDir/config/crypto-config/ordererOrganizations/gtbcsf.com/orderers/$orderer_address_dns/tls/ca.crt
export ORDERER_GENERAL_PROFILE_ENABLED=false
export ORDERER_GENERAL_LISTENADDRESS=0.0.0.0
export ORDERER_GENERAL_LISTENPORT=7050
export ORDERER_GENERAL_GENESISMETHOD=file
export ORDERER_GENERAL_GENESISFILE=$rootDir/config/channel-artifacts/genesis.block
export ORDERER_GENERAL_LOCALMSPDIR=$rootDir/config/crypto-config/ordererOrganizations/gtbcsf.com/orderers/$orderer_address_dns/msp
export ORDERER_GENERAL_LOCALMSPID=OrdererMSP
export ORDERER_FILELEDGER_LOCATION=$rootDir/data/orderer

nohup orderer > $root_dir/log/orderer.log 2>&1 &
