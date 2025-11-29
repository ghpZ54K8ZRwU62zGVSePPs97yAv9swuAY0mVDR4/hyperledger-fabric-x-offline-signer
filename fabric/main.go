package main

import (
	// golang
	"bufio"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	// fabric-gateway
	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/hash"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	// Scan peer tls ca certificate from stdin.
	tlsCaInMsg := "Enter tls ca certificate of the peer: "
	tlsCaOutMsg := "The tls ca certificate of the peer is: "
	tlsCaCert := scanOnStdin(tlsCaInMsg, tlsCaOutMsg)
	fmt.Println(tlsCaCert)

	// Scan msp signcerts of identity from stdin.
	signcertInMsg := "Enter msp sign certificate for the identity: "
	signcertOutMsg := "The msp sign certificate for the identity is: "
	certPemString := scanOnStdin(signcertInMsg, signcertOutMsg)
	fmt.Println(certPemString)

	// Read msp id from stdin.
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter target grpc connection (e.g. localhost:7051): ")
	targetConn, err := reader.ReadString('\n')
	targetConnPlain := strings.TrimSuffix(targetConn, "\n")
	panicOnError(err)
	fmt.Printf("The target grpc connection is %s\n", targetConn)

	// Read msp id from stdin.
	reader = bufio.NewReader(os.Stdin)
	fmt.Print("Enter msp id (e.g. Org1MSP): ")
	mspId, err := reader.ReadString('\n')
	panicOnError(err)
	fmt.Printf("The msp id is %s\n", mspId)

	// Create gRPC client connection, which should be shared by all gateway grpc connections to this endpoint.
	clientConnection, err := NewGrpcConnectionFromTlsCaCert(tlsCaCert, targetConnPlain)
	panicOnError(err)
	defer clientConnection.Close()

	id := NewX509IdentityFromCertMsp([]byte(certPemString), mspId)

	// Create a gateway grpc connection for a specific client identity.
	gateway, err := client.Connect(id, client.WithHash(hash.NONE),
		client.WithClientConnection(clientConnection))
	panicOnError(err)
	defer gateway.Close()

	// Read channel name from stdin.
	reader = bufio.NewReader(os.Stdin)
	fmt.Print("Enter channel name: ")
	channelName, err := reader.ReadString('\n')
	panicOnError(err)
	fmt.Printf("The channel name is %s\n", channelName)

	// Obtain the network.
	network := gateway.GetNetwork(channelName)

	// Read chaincode name from stdin.
	reader = bufio.NewReader(os.Stdin)
	fmt.Print("Enter chaincode name: ")
	chaincodeName, err := reader.ReadString('\n')
	panicOnError(err)
	fmt.Printf("The channel name is %s\n", chaincodeName)

	// Obtain smart contract deployed on the network.
	contract := network.GetContract(chaincodeName)

	// Read transaction name from stdin.
	reader = bufio.NewReader(os.Stdin)
	fmt.Print("Enter transaction name: ")
	txName, err := reader.ReadString('\n')
	panicOnError(err)
	fmt.Printf("The transaction name is %s\n", txName)

	// Read tx proposal arguments from stdin.
	reader = bufio.NewReader(os.Stdin)
	fmt.Println("Enter space-separated transaction args:")
	str, _ := reader.ReadString('\n')
	txArgs := strings.Fields(str)
	fmt.Println("The transaction args is:\n", txArgs)

	// Create a transaction proposal.
	unsignedProposal, err := contract.NewProposal(txName, client.WithArguments(txArgs...))
	panicOnError(err)

	// Off-line sign the proposal.
	proposalBytes, err := unsignedProposal.Bytes()
	panicOnError(err)
	proposalDigest := unsignedProposal.Digest()
	fmt.Printf("The proposal digest is %s. Use it to make a signature offline\n", proposalDigest)
	err = os.WriteFile("./out/proposal.digest", proposalDigest, 0644)
	panicOnError(err)

	// Read proposal signature from stdin.
	reader = bufio.NewReader(os.Stdin)
	fmt.Print("Enter proposal signature: ")
	proposalSig, err := reader.ReadString('\n')
	panicOnError(err)
	fmt.Printf("The proposal signature is %s\n", proposalSig)

	// Construct the signed proposal.
	signedProposal, err := gateway.NewSignedProposal(proposalBytes, []byte(proposalSig))
	panicOnError(err)

	// Endorse the signed proposal to create an endorsed unsigned transaction.
	unsignedTransaction, err := signedProposal.Endorse()
	panicOnError(err)

	// Off-line sign the transaction.
	transactionBytes, err := unsignedTransaction.Bytes()
	panicOnError(err)
	transactionDigest := unsignedTransaction.Digest()
	fmt.Printf("The transaction digest is %s. Use it to make a signature offline\n", transactionDigest)
	err = os.WriteFile("./out/transaction.digest", transactionDigest, 0644)
	panicOnError(err)

	// Read transaction signature from stdin.
	reader = bufio.NewReader(os.Stdin)
	fmt.Print("Enter transaction signature: ")
	transactionSig, err := reader.ReadString('\n')
	panicOnError(err)
	fmt.Printf("The transaction signature is %s\n", transactionSig)

	// Construct the signed transaction.
	signedTransaction, err := gateway.NewSignedTransaction(transactionBytes, []byte(transactionSig))
	panicOnError(err)

	// Submit the signed transaction to the orderering service.
	unsignedCommit, err := signedTransaction.Submit()
	panicOnError(err)

	// Off-line sign the transaction commit status request
	commitBytes, err := unsignedCommit.Bytes()
	panicOnError(err)
	commitDigest := unsignedCommit.Digest()
	fmt.Printf("The commit digest is %s. Use it to make a signature offline\n", commitDigest)
	err = os.WriteFile("./out/commit.digest", commitDigest, 0644)
	panicOnError(err)

	// Read commit signature from stdin.
	reader = bufio.NewReader(os.Stdin)
	fmt.Print("Enter commit signature: ")
	commitSig, err := reader.ReadString('\n')
	panicOnError(err)
	fmt.Printf("The commit signature is %s\n", commitSig)

	// Construct the signed commit.
	signedCommit, err := gateway.NewSignedCommit(commitBytes, []byte(commitSig))
	panicOnError(err)

	// Wait for transaction commit.
	status, err := signedCommit.Status()
	panicOnError(err)
	if !status.Successful {
		panic(fmt.Errorf("transaction %s failed to commit with status code %d", status.TransactionID, int32(status.Code)))
	}
}

// NewIdentityFromCertMsp creates a client identity for the grpc connection using a remotely input X.509 certificatePEM byte and mspID string.
func NewX509IdentityFromCertMsp(certificatePEM []byte, mspID string) *identity.X509Identity {
	certificate, err := identity.CertificateFromPEM(certificatePEM)
	panicOnError(err)

	id, err := identity.NewX509Identity(mspID, certificate)
	panicOnError(err)

	return id
}

// NewGrpcConnectionFromTlsCaCert creates a new gRPC client connection from tlsCaCert string and targetConn string.
func NewGrpcConnectionFromTlsCaCert(tlsCaRootCert string, targetConn string) (*grpc.ClientConn, error) {
	tlsCertificate, err := identity.CertificateFromPEM([]byte(tlsCaRootCert))
	panicOnError(err)

	certPool := x509.NewCertPool()
	certPool.AddCert(tlsCertificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, "")

	// example: dns:///gateway.example.org:1337
	return grpc.NewClient(targetConn, grpc.WithTransportCredentials(transportCredentials))
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func scanOnStdin(inputMsg string, outMsg string) string {
	fmt.Println(inputMsg)
	scanner := bufio.NewScanner(os.Stdin)

	var lines []string
	for {
		scanner.Scan()
		line := scanner.Text()
		if len(line) == 0 {
			break
		}
		lines = append(lines, line)
	}

	err := scanner.Err()
	if err != nil {
		panicOnError(err)
	}

	fmt.Println(outMsg)
	return strings.Join(lines, "\n")
}
