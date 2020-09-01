#!/bin/bash

if [ -z "$1" ] ; then
  name=""
  echo "Hint: You can specify a name for the keypair as argument"
else
  name=$1
fi

echo "Creating the 4096 bit RSA keypair in ${name}hostkey.pem"
openssl genpkey -algorithm RSA -out "${name}hostkey.pem" -pkeyopt rsa_keygen_bits:4096

echo "Extracting public key from keypair, into ${name}pubkey.pem"
openssl rsa -pubout -in "${name}hostkey.pem" -out "${name}pubkey.pem"
