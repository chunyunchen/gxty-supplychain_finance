export CORE_PEER_MSPCONFIGPATH=/Users/ywt/fabric-run-env/config/crypto-config/peerOrganizations/coren.gtbcsf.com/users/Admin@coren.gtbcsf.com/msp
export CORE_PEER_ADDRESS=peer0.coren.gtbcsf.com:7051
export CORE_PEER_LOCALMSPID=CoreEnterpriseMSP
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_TLS_CERT_FILE=/Users/ywt/fabric-run-env/config/crypto-config/peerOrganizations/coren.gtbcsf.com/peers/peer0.coren.gtbcsf.com/tls/server.crt
export CORE_PEER_TLS_KEY_FILE=/Users/ywt/fabric-run-env/config/crypto-config/peerOrganizations/coren.gtbcsf.com/peers/peer0.coren.gtbcsf.com/tls/server.key
export CORE_PEER_TLS_ROOTCERT_FILE=/Users/ywt/fabric-run-env/config/crypto-config/peerOrganizations/coren.gtbcsf.com/peers/peer0.coren.gtbcsf.com/tls/ca.crt

cmd="peer chaincode $* --tls --cafile /Users/ywt/fabric-run-env/config/crypto-config/ordererOrganizations/gtbcsf.com/orderers/orderer.gtbcsf.com/msp/tlscacerts/tlsca.gtbcsf.com-cert.pem"
echo "[COMMAND] $cmd"
eval $cmd
