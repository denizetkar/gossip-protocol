# Get package dependencies.
go get golang.org/x/crypto/scrypt

# Append the current path to the GOPATH.
$Env:GOPATH = $Env:GOPATH + ";" + (get-location)
go build gossip
