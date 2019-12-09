#!/bin/bash

export cur_dir=$(dirname BASH_SOURCE[0])
export root_dir=$(dirname $(cd $cur_dir && pwd))
export script_dir=$root_dir/script

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

export FABRIC_CFG_PATH=$root_dir/config
export CORE_PEER_ID=$peer_host_dns
export CORE_CHAINCODE_MODE=dev
export CORE_PEER_CHAINCODELISTENADDRESS=0.0.0.0:7052
export CORE_PEER_NETWORKID=gxtybcsf
export FABRIC_LOGGING_SPEC=DEBUG
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_TLS_CERT_FILE=$root_dir/config/crypto-config/peerOrganizations/$org_domain/peers/$peer_host_dns/tls/server.crt
export CORE_PEER_TLS_KEY_FILE=$root_dir/config/crypto-config/peerOrganizations/$org_domain/peers/$peer_host_dns/tls/server.key
export CORE_PEER_TLS_ROOTCERT_FILE=$root_dir/config/crypto-config/peerOrganizations/$org_domain/peers/$peer_host_dns/tls/ca.crt
export CORE_PEER_PROFILE_ENABLED=true
export CORE_PEER_ADDRESS=$peer_host_dns:7051
export CORE_PEER_LISTENADDRESS=0.0.0.0:7051
export CORE_PEER_GOSSIP_ENDPOINT=0.0.0.0:7051
export CORE_PEER_EVENTS_ADDRESS=0.0.0.0:7053
export CORE_PEER_LOCALMSPID=$local_msp
export CORE_PEER_MSPCONFIGPATH=$root_dir/config/crypto-config/peerOrganizations/$org_domain/peers/$peer_host_dns/msp
export CORE_PEER_FILESYSTEMPATH=$root_dir/data/peer

nohup peer node start  > $root_dir/log/peer.log 2>&1 &

orderer_host_name=${2:-orderer}
channel_name=$3

orderer=(orderer.gtbcsf.com)
orderer2=(orderer2.gtbcsf.com)
orderer3=(orderer3.gtbcsf.com)
orderer_address_dns=$(eval echo '$'"{$orderer_host_name[0]}")
orderer_tls_ca=$root_dir/config/crypto-config/ordererOrganizations/gtbcsf.com/orderers/$orderer_address_dns/msp/tlscacerts/tlsca.gtbcsf.com-cert.pem

function replaceYAMLFile() {
  # sed on MacOSX does not support -i flag with a null extension. We will use
  # 't' for our back-up's extension and delete it at the end of the function
  ARCH=$(uname -s | grep Darwin)
  if [ "$ARCH" == "Darwin" ]; then
    OPTS="-it"
  else
    OPTS="-i"
  fi

  sed $OPTS "s/FABRIC_CA_DOCKER_COMPOSE_FILE/$(basename $fabric_ca_yaml)/g" $script_dir/start_fabric_ca_container.sh
  # If MacOSX, remove the temporary backup of the docker-compose file
  if [ "$ARCH" == "Darwin" ]; then
    mv $script_dir/start_fabric_ca_container.sht  $script_dir/start_fabric_ca_container.sh
  fi
}

orderer_ca_tls_opt=""
if [[ "$CORE_PEER_TLS_ENABLED" == "true" ]]; then
    orderer_ca_tls_opt="--tls --cafile $orderer_tls_ca"
fi

# generate docker-compose yaml file for fabric-ca-server 
#
##
fabric_ca_yaml="$root_dir/config/docker-compose-fabric-ca.yaml"
peer_org_ca_dir=$root_dir/config/fabric-ca-server-config/$org_domain
ca_priv_key_path_file=$(ls $peer_org_ca_dir/*_sk)
ca_priv_key_file_name=$(basename $ca_priv_key_path_file)
cat > $fabric_ca_yaml << EOF
version: '2'

networks:
  bcsf:
services:
  fabricca:
    image: hyperledger/fabric-ca:latest
    environment:
      - FABRIC_CA_HOME=/etc/hyperledger/fabric-ca-server
      - FABRIC_CA_SERVER_CA_NAME=ca-${org_domain%%.*}
      - FABRIC_CA_SERVER_TLS_ENABLED=true
      - FABRIC_CA_SERVER_TLS_CERTFILE=/etc/hyperledger/fabric-ca-server-config/ca.$org_domain-cert.pem
      - FABRIC_CA_SERVER_TLS_KEYFILE=/etc/hyperledger/fabric-ca-server-config/$ca_priv_key_file_name
    ports:
      - "7054:7054"
    command: sh -c 'fabric-ca-server start --ca.certfile /etc/hyperledger/fabric-ca-server-config/ca.$org_domain-cert.pem --ca.keyfile /etc/hyperledger/fabric-ca-server-config/$ca_priv_key_file_name -b admin:adminpw -d'
    volumes:
      - $peer_org_ca_dir:/etc/hyperledger/fabric-ca-server-config
    container_name: ca_peer${org_domain%%.*}
    networks:
      - bcsf 
EOF

# generate scripts for channel and chaincode operations
#
##

channel_script_name=channel_anchor.sh
chaincode_script_name=peer.sh
copy_script_name=scp_artifacts.sh
restart_script_name=restart_network.sh

# script for channel
cat > ./$channel_script_name << EOF
#!/bin/bash

export FABRIC_CFG_PATH=$root_dir/config
export CORE_PEER_ID=$peer_host_dns
export CORE_PEER_MSPCONFIGPATH=$root_dir/config/crypto-config/peerOrganizations/$org_domain/users/Admin@$org_domain/msp
export CORE_PEER_ADDRESS=$peer_host_dns:7051
export CORE_PEER_LOCALMSPID=$CORE_PEER_LOCALMSPID
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_TLS_CERT_FILE=$CORE_PEER_TLS_CERT_FILE
export CORE_PEER_TLS_KEY_FILE=$CORE_PEER_TLS_KEY_FILE
export CORE_PEER_TLS_ROOTCERT_FILE=$CORE_PEER_TLS_ROOTCERT_FILE

channel_name=$channel_name
peer_host_dns=$peer_host_dns
channle_block_file=$root_dir/config/channel-artifacts/$channel_name.block

DEFAULTREMOTEHOSTs=(
    "btc,10.2.1.91,/home/btc/fabric-run-env"
    "wang,10.2.1.89,/home/wang/fabric-run-env"
)

function scpChannelBlocktoRemoteHosts() {
    OLD_IFS="\$IFS"
    idx=0
    for hoststr in \$@
    do
        IFS=","
        arr=(\$hoststr)
        user=\${arr[0]}
        remoteNode=\${arr[1]}
        remoteDir=\${arr[2]}
        ./scp_artifacts.sh \$user \$remoteNode \$remoteDir
        idx=\$((\$idx+1))
    done
    IFS="\$OLD_IFS"
}

if [[ ! -f \$channle_block_file ]]; then
  cmd="peer channel create -o $orderer_address_dns:7050 -c $channel_name -f $root_dir/config/channel-artifacts/channel.tx --outputBlock $root_dir/config/channel-artifacts/$channel_name.block $orderer_ca_tls_opt"
  echo "[COMMAND] \$cmd"
  eval "\$cmd"
  echo
  if [ \$? -ne 0 ]; then
    echo "ERROR !!!! Unable to create channel: \$channel_name"
    echo
    exit 1
  fi

  scpChannelBlocktoRemoteHosts \${DEFAULTREMOTEHOSTs[@]}
fi

cmd="peer channel join -b \$channle_block_file $orderer_ca_tls_opt"
echo "[COMMAND] \$cmd"
eval "\$cmd"
echo
if [ \$? -ne 0 ]; then
  echo "ERROR !!!! Peer[\$peer_host_dns] unable to join channel[\$channel_name]"
  echo
  exit 1
fi

cmd="peer channel update -o $orderer_address_dns:7050 -c $channel_name -f $root_dir/config/channel-artifacts/${local_msp}_anchors.tx $orderer_ca_tls_opt"
echo "[COMMAND] \$cmd"
eval "\$cmd"
echo
if [ \$? -ne 0 ]; then
  echo "ERROR !!!! Unable to update anchor peer[\$peer_host_dns] channel[\$channel_name]"
  echo
  exit 1
fi

./install_prerequists.sh
echo
EOF

# script for chaincode 
cat > ./$chaincode_script_name << EOF
#!/bin/zsh
## 这里用zsh不用bash，因为zsh可以很好的处理参数中包含单双引号

export FABRIC_CFG_PATH=$root_dir/config
export CORE_PEER_MSPCONFIGPATH=$root_dir/config/crypto-config/peerOrganizations/$org_domain/users/Admin@$org_domain/msp
export CORE_PEER_ADDRESS=$peer_host_dns:7051
export CORE_PEER_LOCALMSPID=$CORE_PEER_LOCALMSPID
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_TLS_CERT_FILE=$CORE_PEER_TLS_CERT_FILE
export CORE_PEER_TLS_KEY_FILE=$CORE_PEER_TLS_KEY_FILE
export CORE_PEER_TLS_ROOTCERT_FILE=$CORE_PEER_TLS_ROOTCERT_FILE

if [[ "\$1" == "list" ]]; then
  cmd="peer chaincode list --installed"
  echo "[COMMAND] \$cmd"
  eval \$cmd
  echo
  cmd="peer chaincode list --instantiated -C $channel_name"
  echo "[COMMAND] \$cmd"
  eval \$cmd
  echo
elif [[ "\$1" == "query" ]]; then
  cmd="peer chaincode \$*"
  peer chaincode \$*
  echo
elif [[ "\$1" == "install" ]]; then
  cmd="peer chaincode \$* $orderer_ca_tls_opt"
  echo "[COMMAND] \$cmd"
  eval \$cmd
  echo
elif [[ "\$1" == "instantiate" ]]; then
  cmd="peer chaincode \$* -o $orderer_address_dns:7050 $orderer_ca_tls_opt"
  echo "[COMMAND] \$cmd"
  peer chaincode \$* -o $orderer_address_dns:7050 $orderer_ca_tls_opt
  if [[ \$? -ne 0 ]]; then
    shift
    cmd="peer chaincode upgrade \$* -o $orderer_address_dns:7050 $orderer_ca_tls_opt"
    echo "[COMMAND] \$cmd"
    echo "Try to upgrade chaincode..."
    peer chaincode upgrade \$* -o $orderer_address_dns:7050 $orderer_ca_tls_opt
    echo
  fi
else
  cmd="peer chaincode \$* $orderer_ca_tls_opt"
  echo "[COMMAND] \$cmd"
  peer chaincode \$* $orderer_ca_tls_opt
fi
EOF

# script for copy files to remote hosts
cat > ./$copy_script_name << EOF
#!/bin/bash

function printHelp() {
    echo "Usange: scp_artifacts <username> <remote host ip> <remote_dir>"
    exit 1
}

if [[ \$# != 3 ]]; then
    printHelp
fi

cur_dir=\$(dirname \$BASH_SOURCE[0])
root_dir=\$(dirname \$(cd \$cur_dir && pwd))

script_dir=\$root_dir/script
config_dir=\$root_dir/config
channel_block_file=\$config_dir/channel-artifacts/$channel_name.block
remote_host="\$1@\$2:\$3"

set -e

if [[ -f \$channel_block_file ]]; then
  echo "copy \$channel_block_file to \$remote_host/config/channel-artifacts"
  scp -r \$channel_block_file \$remote_host/config/channel-artifacts
else
  echo "copy \$script_dir to \$remote_host"
  scp -r \$script_dir \$remote_host

  echo "copy \$config_dir to \$remote_host"
  scp -r \$config_dir \$remote_host
fi
EOF

# script for restart blockchain network
cat > ./$restart_script_name << EOF
#!/bin/bash

# ./bcsf.sh restart -o etcdraft -m SupplierMSP -n orderer3
# ./bcsf.sh restart -o etcdraft -m FinanceMSP -n orderer2
# ./bcsf.sh restart -o etcdraft -m CoreEnterpriseMSP -n orderer

./bcsf.sh restart -o etcdraft -m $local_msp -n $orderer_host_name
EOF

replaceYAMLFile

chmod u+x $chaincode_script_name $channel_script_name $copy_script_name $restart_script_name
