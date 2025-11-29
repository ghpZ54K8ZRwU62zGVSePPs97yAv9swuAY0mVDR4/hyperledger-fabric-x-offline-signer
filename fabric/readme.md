# fabric offline signer

to foster better understand of how fabric-x offline signing is handled ,
a fabric interactive command line interface for offline signing is drafted .

# key materials

a public x.509 certificate ( signcerts ) .
a public tls ca root certificate ( tls/ca ) .
a private key stored in a local drive ( keystore ) .
a grpc connection host:port mapping .
a msp id .

# run

go run .
