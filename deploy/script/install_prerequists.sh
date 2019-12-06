#!/bin/bash
	
#check zsh
function check_zsh() {
	which zsh
	if [[ $? -ne 0 ]]; then
	  echo "Install zsh shell..."
	
	  osn=$(uname -s)
	  if [[ "$osn" == "Darwin" ]]; then
	    sudo brew install zsh
	  else
	    which yum
	    if [[ $? -ne 0 ]]; then
	      sudo apt install zsh
	    else
	      sudo yum install zsh
	    fi
	  fi

      if [[ ! -f /bin/zsh ]]; then
          echo "Make soft link /bin/zsh to /usr/bin/zsh"
          sudo ln -s /usr/bin/zsh /bin/zsh
      fi

	fi
}

#check fabric-ca-server 
function check_fabric_ca_server() {
    which fabric-ca-server
    if [[ $? -ne 0 ]]; then
        echo "Install fabric-ca-server..."
    fi
}

check_zsh
check_fabric_ca_server
