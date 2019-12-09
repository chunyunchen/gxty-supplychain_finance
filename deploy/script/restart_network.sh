#!/bin/bash

# ./bcsf.sh restart -o etcdraft -m SupplierMSP -n orderer3
# ./bcsf.sh restart -o etcdraft -m FinanceMSP -n orderer2
# ./bcsf.sh restart -o etcdraft -m CoreEnterpriseMSP -n orderer

./bcsf.sh restart -o etcdraft -m CoreEnterpriseMSP -n orderer
