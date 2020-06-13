$Env:GOPATH = $Env:GOPATH + ";" + (get-location)
go build gossip
