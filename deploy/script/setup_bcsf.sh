#!/bin/bash

# This script run once when the blockchain network setup
#
# usage: setup_bcsf.sh [username,remotehostIP,remotehostDir [username,remotehostIP,remotehostDir]...]

cur_dir=$(dirname BASH_SOURCE[0])
root_dir=$(dirname $(cd $cur_dir && pwd))
config_dir=$root_dir/config
crypto_config_dir=$config_dir/crypto-config

if [[ -d $crypto_config_dir ]]; then
    echo "Please use bcsf.sh to operate, because the $crypto_config_dir is exist."
    exit 1
fi

LOCAL_MSPs=(
    FinanceMSP
    SupplierMSP
    CoreEnterpriseMSP
)
    
ORDERERHOSTNAMEs=(
    orderer2
    orderer3
    orderer)

# Natively startup orderer and peer
./bcsf.sh up -o etcdraft -m CoreEnterpriseMSP -n orderer

DEFAULTREMOTEHOSTs=(
    "btc,10.2.1.91,/home/btc/fabric-run-env"
    #wang,10.2.1.89,/home/wang/fabric-run-env
)

# Ask user for confirmation to use default remote hosts 
function askDefaultRemoteHosts() {
    read -p "Are you sure use the default remote hosts:(${DEFAULTREMOTEHOSTs[*]}) [Y/n] " ans
  case "$ans" in
  y | Y | "")
    echo "continue..."
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

function remoteStartUp() {
	OLD_IFS="$IFS"
    idx=0
    for hoststr in $@
    do
	    IFS=","
        arr=($hoststr)
        user=${arr[0]}
        remoteNode=${arr[1]}
        remoteDir=${arr[2]}
        ./scp_artifacts.sh $user $remoteNode $remoteDir

#ssh $user@$remoteNode > /dev/null 2>&1 << EOF
#cd $remoteDir 
#./bcsf.sh up -o etcdraft -m ${LOCAL_MSPs[$idx]} -n ${ORDERERHOSTNAMEs[$idx]}
#EOF
        idx=$(($idx+1))
    done
	IFS="$OLD_IFS"
}

if [[ $# -eq 0 ]]; then
    askDefaultRemoteHosts
    remoteStartUp ${DEFAULTREMOTEHOSTs[@]}
else
    remoteStartUp $@
fi

echo [[DONE]]
