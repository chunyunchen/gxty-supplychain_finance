#!/bin/bash

#source /etc/profile
#SHELL=/bin/bash
#PATH=/sbin:/bin:/usr/sbin:/usr/bin:/usr/local/node/bin:/usr/local/bin

function keep_node_app_online()
{
	APP_SCRIPT_ROOT=/home/wang/go/src/github.com/ChainNova/trainingProjects/billEndorse
	ps -ef | grep -v grep | grep -q "node app" || ${APP_SCRIPT_ROOT}/setupFabricNetwork.sh > ${APP_SCRIPT_ROOT}/setup.log 2>&1
}

keep_node_app_online
