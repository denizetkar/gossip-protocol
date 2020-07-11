# Midterm Report

## Assumption Changes
- Change from "Brahms" to "Median-Counter Algorithm"
- Temporary usage of a powershell script for compiling go code, instead of using a make file
- Usage of library 'encoding/gob' instead of protobuf for marshaling/unmarshaling of p2p communication data

## Module Architecture
- The module consists of five logical structures: API Listener, P2P Listener and Central Controller, Membership Controller and Gossiper
	- Central Controller is used to coordinate every task that is done in the other structures
	- Membership Controller manages the peer list and for example sorts out peers that are not online anymore
	- Gossiper is used to manage the gossip data, i.e. the distribution of the data, e.g. requesting or sending data to a peer, or managing notifications to other peers
	- API Listener is used to create API Endpoints for the modules that want to communicate with our Gossip module
	- P2P Listener is used to create P2P Endpoints for the remote peers that want to communicate with our peer
- Each logical structure belongs to a goroutine and is assigned a task by the Central Controller
- The Central Controller will be implemented to be non-blocking by only coordinating and distributing tasks to other structures
- The structures will communicate with the Central Controller with an internal messaging system
- The communication between the peers will be done by implementing an own level 4 protocol which for example includes a Proof of Work concept
	- PoW works by including a cryptographic hash of the handshake message
		- The handshake includes a DH public key, a RSA public key, a timestamp, ip:port and a random nonce
		- The nonce has to be chosen so that the cryptographic hash has a special property
	- PoW will be done by both sides of the communication
- To prevent DoS attacks we also use 'io.LimitedReader' which allows us to read just a limited amount of bytes from the communication buffer as to much data processing could lead to crashes of the system
- If too much data is transmitted an error is return and the connection is terminated
	
## P2P Protocol
- The transmitted p2p messages are encoded using the library 'encoding/gob'
	- Allows to work not on byte level but by using structs
- The transmitted structs consist basically of a message type and a payload, similar to our internal messaging system
```
type P2PMessage struct {
	Type    P2PMessageType
	Payload AnyMessage
}
```

### Message types
- The messages itself can originate either from the gossiper or from the membership controller
- The activities that both control are listed in the following:
- The Gossiper sends at each gossip round a push messages as well as pull requests of rumours to existing peer connections, following the "Median-Counter Algorithm". The Gossipers on the other peers then receive the messages and reply with Pull Replies
- The Membership controller acts in a similar way by sending membership related push/pull messages at every round
- The biggest difference is that the rounds in the membership controller are longer, i.e. 30s vs 500ms.
- Additionally a Disconnect message exists which terminates the connection between two peers 

### Exception handling
- As already discussed, too much transmitted data is handled by terminating the transmission, expecting such a situation only to be maliciously created
- If an Endpoint does not reply within a specific timeout, it is deemed to be down and the connection is terminated exactly as if the peer would send a Disconnect message
- To test for corrupted data we forward the message to the destination module which checks the validity of the data