#!/bin/zsh
## 这里用zsh不用bash，因为zsh可以很好的处理参数中包含单双引号

export CORE_PEER_MSPCONFIGPATH=/Users/ywt/fabric-run-env/config/crypto-config/peerOrganizations/coren.gtbcsf.com/users/Admin@coren.gtbcsf.com/msp
export CORE_PEER_ADDRESS=peer0.coren.gtbcsf.com:7051
export CORE_PEER_LOCALMSPID=CoreEnterpriseMSP
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_TLS_CERT_FILE=/Users/ywt/fabric-run-env/config/crypto-config/peerOrganizations/coren.gtbcsf.com/peers/peer0.coren.gtbcsf.com/tls/server.crt
export CORE_PEER_TLS_KEY_FILE=/Users/ywt/fabric-run-env/config/crypto-config/peerOrganizations/coren.gtbcsf.com/peers/peer0.coren.gtbcsf.com/tls/server.key
export CORE_PEER_TLS_ROOTCERT_FILE=/Users/ywt/fabric-run-env/config/crypto-config/peerOrganizations/coren.gtbcsf.com/peers/peer0.coren.gtbcsf.com/tls/ca.crt

if [[ "$1" == "list" ]]; then
  cmd="peer chaincode list --installed"
  echo "[COMMAND] $cmd"
  eval $cmd
  echo
  cmd="peer chaincode list --instantiated -C sfchl"
  echo "[COMMAND] $cmd"
  eval $cmd
  echo
elif [[ "$1" == "query" ]]; then
  cmd="peer chaincode $*"
  peer chaincode $*
  echo
elif [[ "$1" == "install" ]]; then
  cmd="peer chaincode $* --tls --cafile /Users/ywt/fabric-run-env/config/crypto-config/ordererOrganizations/gtbcsf.com/orderers/orderer.gtbcsf.com/msp/tlscacerts/tlsca.gtbcsf.com-cert.pem"
  echo "[COMMAND] $cmd"
  eval $cmd
  echo
elif [[ "$1" == "instantiate" ]]; then
  cmd="peer chaincode $* -o orderer.gtbcsf.com:7050 --tls --cafile /Users/ywt/fabric-run-env/config/crypto-config/ordererOrganizations/gtbcsf.com/orderers/orderer.gtbcsf.com/msp/tlscacerts/tlsca.gtbcsf.com-cert.pem"
  echo "[COMMAND] $cmd"
  peer chaincode $* -o orderer.gtbcsf.com:7050 --tls --cafile /Users/ywt/fabric-run-env/config/crypto-config/ordererOrganizations/gtbcsf.com/orderers/orderer.gtbcsf.com/msp/tlscacerts/tlsca.gtbcsf.com-cert.pem
  if [[ $? -ne 0 ]]; then
    shift
    cmd="peer chaincode upgrade $* -o orderer.gtbcsf.com:7050 --tls --cafile /Users/ywt/fabric-run-env/config/crypto-config/ordererOrganizations/gtbcsf.com/orderers/orderer.gtbcsf.com/msp/tlscacerts/tlsca.gtbcsf.com-cert.pem"
    echo "[COMMAND] $cmd"
    echo "Try to upgrade chaincode..."
    peer chaincode upgrade $* -o orderer.gtbcsf.com:7050 --tls --cafile /Users/ywt/fabric-run-env/config/crypto-config/ordererOrganizations/gtbcsf.com/orderers/orderer.gtbcsf.com/msp/tlscacerts/tlsca.gtbcsf.com-cert.pem
    echo
  fi
else
  cmd="peer chaincode $* --tls --cafile /Users/ywt/fabric-run-env/config/crypto-config/ordererOrganizations/gtbcsf.com/orderers/orderer.gtbcsf.com/msp/tlscacerts/tlsca.gtbcsf.com-cert.pem"
  echo "[COMMAND] $cmd"
  peer chaincode $* --tls --cafile /Users/ywt/fabric-run-env/config/crypto-config/ordererOrganizations/gtbcsf.com/orderers/orderer.gtbcsf.com/msp/tlscacerts/tlsca.gtbcsf.com-cert.pem
fi
