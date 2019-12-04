#!/bin/bash

export scriptDir=$(dirname BASH_SOURCE[0])
export rootDir=$(dirname $(cd $scriptDir && pwd))

local_msp=$1
if [[ -z $local_msp ]]; then
    echo "Start peer service failed. Please specify peer LocalMSP [FinanceMSP|CoreEnterpriseMSP|SupplierMSP] which is defined in configtx.yaml"
    exit 1
    echo
fi

FinanceMSP=(finance.gtbcsf.com peer0.finance.gtbcsf.com)
CoreEnterpriseMSP=(coren.gtbcsf.com peer0.coren.gtbcsf.com)
SupplierMSP=(supplier.gtbcsf.com peer0.supplier.gtbcsf.com)
peer_host_dns=$(eval echo '$'"{$local_msp[1]}")
org_domain=$(eval echo '$'"{$local_msp[0]}")

export CORE_PEER_ID=$peer_host_dns
export CORE_CHAINCODE_MODE=dev
export CORE_PEER_CHAINCODELISTENADDRESS=0.0.0.0:7052
export CORE_PEER_NETWORKID=gxtybcsf
export CORE_LOGGING_LEVEL=DEBUG
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_TLS_CERT_FILE=$rootDir/config/crypto-config/peerOrganizations/$org_domain/peers/$peer_host_dns/tls/server.crt
export CORE_PEER_TLS_KEY_FILE=$rootDir/config/crypto-config/peerOrganizations/$org_domain/peers/$peer_host_dns/tls/server.key
export CORE_PEER_TLS_ROOTCERT_FILE=$rootDir/config/crypto-config/peerOrganizations/$org_domain/peers/$peer_host_dns/tls/ca.crt
export CORE_PEER_PROFILE_ENABLED=true
export CORE_PEER_ADDRESS=$peer_host_dns:7051
export CORE_PEER_LISTENADDRESS=0.0.0.0:7051
export CORE_PEER_GOSSIP_ENDPOINT=0.0.0.0:7051
export CORE_PEER_EVENTS_ADDRESS=0.0.0.0:7053
export CORE_PEER_LOCALMSPID=$local_msp
export CORE_PEER_MSPCONFIGPATH=$rootDir/config/crypto-config/peerOrganizations/$org_domain/peers/$peer_host_dns/msp
export CORE_PEER_FILESYSTEMPATH=$rootDir/data/peer

nohup peer node start  > $rootDir/log/peer.log 2>&1 &

orderer_host_name=${2:-orderer}
channel_name=$3

orderer=(orderer.gtbcsf.com)
orderer2=(orderer2.gtbcsf.com)
orderer3=(orderer3.gtbcsf.com)
orderer_address_dns=$(eval echo '$'"{$orderer_host_name[0]}")
orderer_tls_ca=$rootDir/config/crypto-config/ordererOrganizations/gtbcsf.com/orderers/$orderer_address_dns/msp/tlscacerts/tlsca.gtbcsf.com-cert.pem

orderer_ca_tls_opt=""
if [[ "$CORE_PEER_TLS_ENABLED" == "true" ]]; then
    orderer_ca_tls_opt="--tls $CORE_PEER_TLS_ENABLED --cafile $orderer_tls_ca"
fi

# generate scripts for channel and chaincode operations
#
# script for channel
cat > ./create_channel_and_update_anchor.sh << EOF
export CORE_PEER_ID=$peer_host_dns
export CORE_PEER_MSPCONFIGPATH=$rootDir/config/crypto-config/peerOrganizations/$org_domain/users/Admin@$org_domain/msp
export CORE_PEER_ADDRESS=$peer_host_dns:7051
export CORE_PEER_LOCALMSPID=$CORE_PEER_LOCALMSPID
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_TLS_CERT_FILE=$CORE_PEER_TLS_CERT_FILE
export CORE_PEER_TLS_KEY_FILE=$CORE_PEER_TLS_KEY_FILE
export CORE_PEER_TLS_ROOTCERT_FILE=$CORE_PEER_TLS_ROOTCERT_FILE

channel_name=$channel_name
peer_host_dns=$peer_host_dns

peer channel create -o $orderer_address_dns:7050 -c $channel_name -f $rootDir/config/channel-artifacts/channel.tx --outputBlock $rootDir/config/channel-artifacts/$channel_name.block $orderer_ca_tls_opt
if [ \$? -ne 0 ]; then
  echo "ERROR !!!! Unable to create channel: \$channel_name"
  exit 1
fi

peer channel join -b $rootDir/config/channel-artifacts/$channel_name.block $orderer_ca_tls_opt
if [ \$? -ne 0 ]; then
  echo "ERROR !!!! Peer[\$peer_host_dns] unable to join channel[\$channel_name]"
  exit 1
fi

peer channel update -o $orderer_address_dns:7050 -c $channel_name -f $rootDir/config/channel-artifacts/${local_msp}_anchors.tx $orderer_ca_tls_opt
if [ \$? -ne 0 ]; then
  echo "ERROR !!!! Unable to update anchor peer[\$peer_host_dns] channel[\$channel_name]"
  exit 1
fi
echo
EOF

# script for chaincode 
cat > ./peer.sh << EOF
export CORE_PEER_MSPCONFIGPATH=$rootDir/config/crypto-config/peerOrganizations/$org_domain/users/Admin@$org_domain/msp
export CORE_PEER_ADDRESS=$peer_host_dns:7051
export CORE_PEER_LOCALMSPID=$CORE_PEER_LOCALMSPID
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_TLS_CERT_FILE=$CORE_PEER_TLS_CERT_FILE
export CORE_PEER_TLS_KEY_FILE=$CORE_PEER_TLS_KEY_FILE
export CORE_PEER_TLS_ROOTCERT_FILE=$CORE_PEER_TLS_ROOTCERT_FILE

peer chaincode \$* $orderer_ca_tls_opt
EOF

chmod u+x peer.sh create_channel_and_update_anchor.sh
