#!/bin/bash

export CORE_PEER_ID=peer0.coren.gtbcsf.com
export CORE_PEER_MSPCONFIGPATH=/Users/ywt/fabric-run-env/config/crypto-config/peerOrganizations/coren.gtbcsf.com/users/Admin@coren.gtbcsf.com/msp
export CORE_PEER_ADDRESS=peer0.coren.gtbcsf.com:7051
export CORE_PEER_LOCALMSPID=CoreEnterpriseMSP
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_TLS_CERT_FILE=/Users/ywt/fabric-run-env/config/crypto-config/peerOrganizations/coren.gtbcsf.com/peers/peer0.coren.gtbcsf.com/tls/server.crt
export CORE_PEER_TLS_KEY_FILE=/Users/ywt/fabric-run-env/config/crypto-config/peerOrganizations/coren.gtbcsf.com/peers/peer0.coren.gtbcsf.com/tls/server.key
export CORE_PEER_TLS_ROOTCERT_FILE=/Users/ywt/fabric-run-env/config/crypto-config/peerOrganizations/coren.gtbcsf.com/peers/peer0.coren.gtbcsf.com/tls/ca.crt

channel_name=sfchl
peer_host_dns=peer0.coren.gtbcsf.com
channle_block_file=/Users/ywt/fabric-run-env/config/channel-artifacts/sfchl.block

if [[ ! -f $channle_block_file ]]; then
  cmd="peer channel create -o orderer.gtbcsf.com:7050 -c sfchl -f /Users/ywt/fabric-run-env/config/channel-artifacts/channel.tx --outputBlock /Users/ywt/fabric-run-env/config/channel-artifacts/sfchl.block --tls --cafile /Users/ywt/fabric-run-env/config/crypto-config/ordererOrganizations/gtbcsf.com/orderers/orderer.gtbcsf.com/msp/tlscacerts/tlsca.gtbcsf.com-cert.pem"
  echo "[COMMAND] $cmd"
  eval "$cmd"
  echo
  if [ $? -ne 0 ]; then
    echo "ERROR !!!! Unable to create channel: $channel_name"
    echo
    exit 1
  fi
fi

cmd="peer channel join -b $channle_block_file --tls --cafile /Users/ywt/fabric-run-env/config/crypto-config/ordererOrganizations/gtbcsf.com/orderers/orderer.gtbcsf.com/msp/tlscacerts/tlsca.gtbcsf.com-cert.pem"
echo "[COMMAND] $cmd"
eval "$cmd"
echo
if [ $? -ne 0 ]; then
  echo "ERROR !!!! Peer[$peer_host_dns] unable to join channel[$channel_name]"
  echo
  exit 1
fi

cmd="peer channel update -o orderer.gtbcsf.com:7050 -c sfchl -f /Users/ywt/fabric-run-env/config/channel-artifacts/CoreEnterpriseMSP_anchors.tx --tls --cafile /Users/ywt/fabric-run-env/config/crypto-config/ordererOrganizations/gtbcsf.com/orderers/orderer.gtbcsf.com/msp/tlscacerts/tlsca.gtbcsf.com-cert.pem"
echo "[COMMAND] $cmd"
eval "$cmd"
echo
if [ $? -ne 0 ]; then
  echo "ERROR !!!! Unable to update anchor peer[$peer_host_dns] channel[$channel_name]"
  echo
  exit 1
fi

./install_prerequists.sh
echo
