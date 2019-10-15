#!/bin/bash
#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

script_root=$(dirname ${BASH_SOURCE[0]})

function dkcl(){
        CONTAINER_IDS=$(docker ps -aq)
	echo
        if [ -z "$CONTAINER_IDS" -o "$CONTAINER_IDS" = " " ]; then
                echo "========== No containers available for deletion =========="
        else
                docker rm -f $CONTAINER_IDS
        fi
	echo
}

function dkrm(){
        DOCKER_IMAGE_IDS=$(docker images | grep "dev\|none\|test-vp\|peer[0-9]-" | awk '{print $3}')
	echo
        if [ -z "$DOCKER_IMAGE_IDS" -o "$DOCKER_IMAGE_IDS" = " " ]; then
		echo "========== No images available for deletion ==========="
        else
                docker rmi -f $DOCKER_IMAGE_IDS
        fi
	echo
}

function restartNetwork() {
	echo

        #teardown the network and clean the containers and intermediate images
	cd ${script_root}/artifacts
	docker-compose down
	dkcl
	dkrm

	#Cleanup the material
	rm -rf /tmp/hfc-test-kvs_peerOrg* $HOME/.hfc-key-store/ /tmp/fabric-client-kvs_peerOrg*

	#Start the network
	docker-compose up -d
	echo
}

function installNodeModules() {
	cd $script_root

	if [ -d ${script_root}/node_modules ]; then
		echo "============== node modules installed already ============="
	else
		echo "============== Installing node modules ============="
		/usr/local/node/bin/npm install
	fi
	echo
}

function start_node_app()
{
	cd $script_root
	PORT=4000 nohup /usr/local/node/bin/node app >> setup.log &
}

restartNetwork 
installNodeModules
start_node_app
