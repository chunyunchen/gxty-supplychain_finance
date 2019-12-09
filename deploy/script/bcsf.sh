#!/bin/bash

cur_dir=$(dirname $BASH_SOURCE[0])
script_dir=$(cd $cur_dir && pwd)
root_dir=$(dirname $script_dir)
artifacts_dir=$root_dir/config/channel-artifacts
crypto_config_dir=$root_dir/config/crypto-config
fabric_ca_server_config=$root_dir/config/fabric-ca-server-config
del_flag_file=$crypto_config_dir/.del_flag

FinanceMSP=(finance.gtbcsf.com peer0.finance.gtbcsf.com)
CoreEnterpriseMSP=(coren.gtbcsf.com peer0.coren.gtbcsf.com)
SupplierMSP=(supplier.gtbcsf.com peer0.supplier.gtbcsf.com)

export FABRIC_CFG_PATH=${root_dir}/config

# Add a flag to avoid accidently delete the Certs artifacts
function copyOrgCACertstoFabricCA() {
  peerOrgCACertsDir=$crypto_config_dir/peerOrganizations 
  for local_msp in $@
  do
    org_domain=$(eval echo '$'"{$local_msp[0]}")
    mkdir -p $fabric_ca_server_config/$org_domain
    cp $peerOrgCACertsDir/$org_domain/ca/* $fabric_ca_server_config/$org_domain
  done
}

# Create configure and data dirs
function create_dirs() {
  if [[ ! -d $root_dir/config/fabric-ca-server-config ]]; then
    mkdir -p $root_dir/config/fabric-ca-server-config
  fi      

  if [[ ! -d $root_dir/data ]]; then
    mkdir -p $root_dir/data/orderer $root_dir/data/peer
  fi      

  if [[ ! -d $root_dir/log ]]; then
    mkdir -p $root_dir/log
  fi      

  if [[ ! -d $artifacts_dir ]]; then
    mkdir -p $artifacts_dir
  fi      
}

# Add a flag to avoid accidently delete the Certs artifacts
function addDeleteFlag() {
  if [[ "$(uname -s)" == "Darwin" ]]; then
    echo "$(uname -n)" > $del_flag_file
  else
    echo "$(hostid)" > $del_flag_file  
  fi

  if [ $res -ne 0 ]; then
    echo "Failed to add delete-flag..."
    exit 1
  fi
}

function checkDeleteFlag() {
  delFlag="$(cat $del_flag_file)"
  hostFlag=""

  if [[ "$(uname -s)" == "Darwin" ]]; then
    hostFlag="$(uname -n)"
  else
    hostFlag="$(hostid)"
  fi

  if [[ "$delFlag" != "$hostFlag" ]]; then
      echo "Can't delete $root_dir/config, because you are not the author"
      exit 1
  fi
}

# Delete data and configs for orderer and peer
function delete_dirs() {
    rm -rf $artifacts_dir
    rm -rf $root_dir/data
    rm -rf $crypto_config_dir
    rm -rf $root_dir/config/fabric-ca-server-config
}

# Delete only data for orderer and peer
function delete_data_dirs() {
    rm -rf $root_dir/data
}

# Print the usage message
function printHelp() {
  echo "Usage: "
  echo "  bcsf.sh <mode> [-c <channel name>] [-t <timeout>] [-d <delay>] [-s <dbtype>] [-l <language>] [-o <consensus-type>] [-v]"
  echo "    <mode> - one of 'up', 'down', 'restart', 'cleanup', 'deldataup' or 'generate'"
  echo "      - 'up' - bring up the network"
  echo "      - 'cleanup' - delete data and configs for orderer and peer"
  echo "      - 'deldataup' - delete only data for orderer and peer"
  echo "      - 'down' - stop the network"
  echo "      - 'restart' - restart the network"
  echo "      - 'generate' - generate required certificates and genesis block"
  echo "    -c <channel name> - channel name to use (defaults to \"mychannel\")"
  echo "    -t <timeout> - CLI timeout duration in seconds (defaults to 10)"
  echo "    -d <delay> - delay duration in seconds (defaults to 3)"
  echo "    -s <dbtype> - the database backend to use: goleveldb (default) or couchdb"
  echo "    -l <language> - the chaincode language: golang (default) or node"
  echo "    -o <consensus-type> - the consensus-type of the ordering service: solo (default), kafka, or etcdraft"
  echo "    -n <orderer hostname> - the orderer hostname: orderer(default)"
  echo "    -m <localmsp> - the peer local msp: CoreEnterpriseMSP(default)"
  echo "    -v - verbose mode"
  echo "  bcsf.sh -h (print this message)"
  echo
  echo "Typically, one would first generate the required certificates and "
  echo "genesis block, then start up the blockchain network. e.g.:"
  echo
  echo "        bcsf.sh generate -c sfchannel"
  echo "        bcsf.sh up -c sfchannel -s couchdb"
  echo "        bcsf.sh up -c sfchannel -s couchdb"
  echo "        bcsf.sh up -l node"
  echo "        bcsf.sh down -c sfchannel"
  echo
  echo "Taking all defaults:"
  echo "        bcsf.sh generate"
  echo "        bcsf.sh up"
  echo "        bcsf.sh down"
}

# Ask user for confirmation to delete all data 
function askDeleteData() {
  read -p "!!!Be careful!!!  You will delete $* for orderer and peer! Are you sure detete data to startup network? [Y/n] " ans
  case "$ans" in
  y | Y | "")
    echo "deleting ..."
    ;;
  n | N)
    echo "exiting..."
    exit 1
    ;;
  *)
    echo "invalid response"
    askDeleteData
    ;;
  esac
}

# Ask user for confirmation to proceed
function askProceed() {
  read -p "Continue? [Y/n] " ans
  case "$ans" in
  y | Y | "")
    echo "proceeding ..."
    ;;
  n | N)
    echo "exiting..."
    exit 1
    ;;
  *)
    echo "invalid response"
    askProceed
    ;;
  esac
}

# Generate the needed certificates, the genesis block and start the network.
function networkUp() {
  #checkPrereqs
  # generate artifacts if they don't exist
  if [ ! -d "$crypto_config_dir" ]; then
    generateCerts
    copyOrgCACertstoFabricCA CoreEnterpriseMSP FinanceMSP SupplierMSP
    generateChannelArtifacts
  fi

  $script_dir/start_orderer.sh $ORDERER_HOST_NAME
  if [ $? -ne 0 ]; then
    echo "ERROR !!!! Unable to start orderer"
    exit 1
  fi

  $script_dir/start_couchdb_container.sh
  if [ $? -ne 0 ]; then
    echo "ERROR !!!! Unable to start couchdb service"
    exit 1
  fi

  $script_dir/start_peer.sh $LOCAL_MSP $ORDERER_HOST_NAME $CHANNEL_NAME
  if [ $? -ne 0 ]; then
    echo "ERROR !!!! Unable to start peer"
    exit 1
  fi

  if [ "$CONSENSUS_TYPE" == "kafka" ]; then
    sleep 1
    echo "Sleeping 10s to allow $CONSENSUS_TYPE cluster to complete booting"
    sleep 9
  fi

#  if [ "$CONSENSUS_TYPE" == "etcdraft" ]; then
#    sleep 1
#    echo "Sleeping 15s to allow $CONSENSUS_TYPE cluster to complete booting"
#    sleep 14
#  fi

  # now run the end to end script
  #docker exec cli scripts/script.sh $CHANNEL_NAME $CLI_DELAY $LANGUAGE $CLI_TIMEOUT $VERBOSE
  if [ $? -ne 0 ]; then
    echo "ERROR !!!! Test failed"
    exit 1
  fi
}

# Tear down running network
function networkDown() {
  $script_dir/stop_orderer.sh
  $script_dir/stop_peer.sh
}

# replace constants with private key file names generated by the cryptogen tool
function replacePrivateKey() {
  # sed on MacOSX does not support -i flag with a null extension. We will use
  # 't' for our back-up's extension and delete it at the end of the function
  ARCH=$(uname -s | grep Darwin)
  if [ "$ARCH" == "Darwin" ]; then
    OPTS="-it"
  else
    OPTS="-i"
  fi

  # Copy the template to the file that will be modified to add the private key
  cp docker-compose-e2e-template.yaml docker-compose-e2e.yaml

  # The next steps will replace the template's contents with the
  # actual values of the private key file names for the two CAs.
  CURRENT_DIR=$PWD
  cd crypto-config/peerOrganizations/org1.example.com/ca/
  PRIV_KEY=$(ls *_sk)
  cd "$CURRENT_DIR"
  sed $OPTS "s/CA1_PRIVATE_KEY/${PRIV_KEY}/g" docker-compose-e2e.yaml
  cd crypto-config/peerOrganizations/org2.example.com/ca/
  PRIV_KEY=$(ls *_sk)
  cd "$CURRENT_DIR"
  sed $OPTS "s/CA2_PRIVATE_KEY/${PRIV_KEY}/g" docker-compose-e2e.yaml
  # If MacOSX, remove the temporary backup of the docker-compose file
  if [ "$ARCH" == "Darwin" ]; then
    rm docker-compose-e2e.yamlt
  fi
}


# Generates Org certs using cryptogen tool
function generateCerts() {
  which cryptogen
  if [ "$?" -ne 0 ]; then
    echo "cryptogen tool not found. exiting"
    exit 1
  fi
  echo
  echo "##########################################################"
  echo "##### Generate certificates using cryptogen tool #########"
  echo "##########################################################"

  if [ -d "$root_dir/config/crypto-config" ]; then
    rm -Rf $root_dir/config/crypto-config
  fi
  set -x
  cryptogen generate --config=$root_dir/config/crypto-config.yaml --output=$root_dir/config/crypto-config
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to generate certificates..."
    exit 1
  fi
  
  addDeleteFlag
  echo
}

function generateChannelConfiguration() {
  channel_config=$1
  shift
  all_orgs=$@  

  echo
  echo "#################################################################"
  echo "### Generating channel configuration transaction 'channel.tx' ###"
  echo "#################################################################"
  set -x
  configtxgen -profile $channel_config -outputCreateChannelTx $artifacts_dir/channel.tx -channelID $CHANNEL_NAME
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to generate channel configuration transaction..."
    exit 1
  fi

  for org in $all_orgs
  do
    echo
    echo "#################################################################"
    echo "#######   Generating anchor peer update for Org $org   ##########"
    echo "#################################################################"
    set -x
    configtxgen -profile $channel_config -outputAnchorPeersUpdate $artifacts_dir/${org}_anchors.tx -channelID $CHANNEL_NAME -asOrg $org
    res=$?
    set +x
    if [ $res -ne 0 ]; then
      echo "Failed to generate anchor peer update for Org1MSP..."
      exit 1
    fi
  done
  echo
}

# Generate orderer genesis block, channel configuration transaction and
# anchor peer update transactions
function generateChannelArtifacts() {
  which configtxgen
  if [ "$?" -ne 0 ]; then
    echo "configtxgen tool not found. exiting"
    exit 1
  fi

  echo "##########################################################"
  echo "#########  Generating Orderer Genesis block ##############"
  echo "##########################################################"
  # Note: For some unknown reason (at least for now) the block file can't be
  # named orderer.genesis.block or the orderer will fail to launch!
  echo "CONSENSUS_TYPE="$CONSENSUS_TYPE
  set -x
  tx_opts="-channelID bcsf-sys-channel -outputBlock $artifacts_dir/genesis.block -configPath $root_dir/config"
  if [ "$CONSENSUS_TYPE" == "solo" ]; then
    configtxgen -profile SoloOrdererGenesis $tx_opts
  elif [ "$CONSENSUS_TYPE" == "kafka" ]; then
    configtxgen -profile KafkaOrdererGenesis $tx_opts
  elif [ "$CONSENSUS_TYPE" == "etcdraft" ]; then
    configtxgen -profile EtcdRaftOrdererGenesis $tx_opts
  else
    set +x
    echo "unrecognized CONSESUS_TYPE='$CONSENSUS_TYPE'. exiting"
    exit 1
  fi
  res=$?
  set +x
  if [ $res -ne 0 ]; then
    echo "Failed to generate orderer genesis block..."
    exit 1
  fi
  if [ "$CONSENSUS_TYPE" == "solo" ]; then
    generateChannelConfiguration OneOrgChannel CoreEnterpriseMSP
  else
    generateChannelConfiguration ThreeOrgsChannel CoreEnterpriseMSP FinanceMSP SupplierMSP
  fi
  echo
}

# timeout duration - the duration the CLI should wait for a response from
# another container before giving up
CLI_TIMEOUT=10
# default for delay between commands
CLI_DELAY=3
# channel name defaults to "mychannel"
CHANNEL_NAME="sfchl"
# use golang as the default language for chaincode
LANGUAGE=golang
# default image tag
IMAGETAG="latest"
# default consensus type
CONSENSUS_TYPE="solo"
# default orderer hostname 
ORDERER_HOST_NAME=orderer
# default peer local msp
LOCAL_MSP=CoreEnterpriseMSP

MODE=$1
shift
# Determine whether starting, stopping, restarting, generating or upgrading
if [ "$MODE" == "up" ]; then
  EXPMODE="Starting"
elif [ "$MODE" == "cleanup" ]; then
  EXPMODE="Delete data and configs for orderer and peer then starting"
elif [ "$MODE" == "deldataup" ]; then
  EXPMODE="Delete only data for orderer and peer then starting"
elif [ "$MODE" == "down" ]; then
  EXPMODE="Stopping"
elif [ "$MODE" == "restart" ]; then
  EXPMODE="Restarting"
elif [ "$MODE" == "generate" ]; then
  EXPMODE="Generating certs and genesis block"
elif [ "$MODE" == "upgrade" ]; then
  EXPMODE="Upgrading the network"
else
  printHelp
  exit 1
fi

while getopts "h?c:t:d:s:l:o:m:n:v" opt; do
  case "$opt" in
  h | \?)
    printHelp
    exit 0
    ;;
  c)
    CHANNEL_NAME=$OPTARG
    ;;
  t)
    CLI_TIMEOUT=$OPTARG
    ;;
  d)
    CLI_DELAY=$OPTARG
    ;;
  s)
    IF_COUCHDB=$OPTARG
    ;;
  l)
    LANGUAGE=$OPTARG
    ;;
  o)
    CONSENSUS_TYPE=$OPTARG
    ;;
  m)
    LOCAL_MSP=$OPTARG
    ;;
  n)
    ORDERER_HOST_NAME=$OPTARG
    ;;
  v)
    VERBOSE=true
    ;;
  esac
done

# Announce what was requested

if [ "${IF_COUCHDB}" == "couchdb" ]; then
  echo
  echo "${EXPMODE} for channel '${CHANNEL_NAME}' with CLI timeout of '${CLI_TIMEOUT}' seconds and CLI delay of '${CLI_DELAY}' seconds and using database '${IF_COUCHDB}'"
else
  echo "${EXPMODE} for channel '${CHANNEL_NAME}' with CLI timeout of '${CLI_TIMEOUT}' seconds and CLI delay of '${CLI_DELAY}' seconds"
fi

# ask for confirmation to proceed
askProceed

if [ "${MODE}" == "deldataup" ]; then
  askDeleteData "data"
  delete_data_dirs
fi

if [ "${MODE}" == "cleanup" ]; then
  checkDeleteFlag
  askDeleteData "data and configs"
  delete_dirs
fi

# create required directories
create_dirs

#Create the network using docker compose
if [ "${MODE}" == "up" ]; then
  networkUp
elif [ "${MODE}" == "down" ]; then ## Clear the network
  networkDown
elif [ "${MODE}" == "generate" ]; then ## Generate Artifacts
  generateCerts
  copyOrgCACertstoFabricCA CoreEnterpriseMSP FinanceMSP SupplierMSP
  generateChannelArtifacts
elif [ "${MODE}" == "restart" -o "${MODE}" == "cleanup" -o "${MODE}" == "deldataup" ]; then ## Restart the network
  networkDown
  networkUp
elif [ "${MODE}" == "upgrade" ]; then ## Upgrade the network from version 1.2.x to 1.3.x
  upgradeNetwork
else
  printHelp
  exit 1
fi

echo [NATIVE DONE]
