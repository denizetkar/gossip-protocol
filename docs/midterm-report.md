# Midterm Report

## Assumption Changes
- Change from "Brahms" to "Median-Counter Algorithm"
- Temporary usage of a powershell script for compiling go code, instead of using a make file
- Usage of library 'encoding/gob' instead of protobuf for marshalling/unmarshalling of p2p communication data

## Module Architecture
- The module consists of five logical structures: API Listener, P2P Listener and Central Controller, Membership Controller and Gossiper
	- Central Controller is used to coordinate every task that is done in the other structures
	- Membership Controller manages the peer list and for example sorts out peers that are not online anymore
	- Gossiper is used to manage the Gossip Protocol, e.g. managing notifications to other peers
	- API Listener is used to create API Endpoints for the modules that want to communicate with our Gossip module
	- P2P Listener is used to create P2P Endpoints to other peers
- Each logical structure belongs to a goroutine and is assigned a task by the Central Controller
- The Central Controller will be implemented to be non-blocking by only coordinating and distributing tasks to other structures
- The structures will communicate with the Central Controller with an internal messaging system
- The communication between the peers will be done by implementing an own level 4 protocol which for example includes a Proof of Work concept
	- PoW works by including a cryptographic hash of the handshake message
		- The handshake includes a DH public key, a RSA public key, a timestamp, ip:port and a random nonce
		- The nonce has to be chosen so that the cryptographic hash has a special property
	- PoW will be done for only on the client side of the communication