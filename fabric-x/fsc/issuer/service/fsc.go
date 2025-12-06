// SPDX-License-Identifier: MIT

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package service

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/node"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/services/logging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/endpoint"
	viewregistry "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

var logger = logging.MustGetLogger() // TODO

type FabricSmartClient struct {
	node *node.Node
}

func NewFSC(node *node.Node) *FabricSmartClient {
	return &FabricSmartClient{node: node}
}

// Amount The amount to issue, transfer or redeem.
type Amount struct {
	// Code the code of the token
	Code string

	// Value value in base units (usually cents)
	Value uint64
}

var (
	ErrWalletNotFound = errors.New("wallet not found")
	ErrBalance        = errors.New("error getting balance")
	ErrTechnicalError = errors.New("server error")
)

// IssueOnSignature prepares issuing an amount of tokens to a wallet. It prepares the transaction,
// get the token request of the transaction for signatures of an offline signer.
func (f FabricSmartClient) IssueOnSignature(ctx context.Context) (*IssueOnSignatureResponse, error) {
	logger.Infof("going to construct a tx for offline signing")
	mgr, err := viewregistry.GetManager(f.node)
	if err != nil {
		return nil, err
	}
	res, err := mgr.InitiateView(&IssueOnSignatureView{}, ctx)
	if err != nil {
		logger.Errorf("error issuing: %s", err.Error())
		return nil, err
	}
	issueOnSignatureRes, ok := res.(*IssueOnSignatureResponse)
	if !ok {
		return nil, errors.New("cannot parse issue on signature response")
	}
	logger.Infof("prepared a base64 encoded tx for issue [%s] with a base64 encoded message for sign: [%s]", issueOnSignatureRes.EncodedTx, issueOnSignatureRes.EncodedSignMessage)
	return issueOnSignatureRes, nil
}

// IssueToken issues an amount of tokens to a wallet from an offline signature and transaction in raw. It connects to the other node,
// prepares the transaction and sends it to the blockchain for endorsement and commit.
func (f FabricSmartClient) IssueToken(ctx context.Context, tokenType string, quantity uint64, recipient string, recipientNode string, message string, signature string, transaction string) (string, error) {
	logger.Infof("going to issue %d %s to [%s] on [%s] with message [%s], signature [%s], and transaction [%s]", quantity, tokenType, recipient, recipientNode, message, signature, transaction)
	mgr, err := viewregistry.GetManager(f.node)
	if err != nil {
		return "", err
	}
	res, err := mgr.InitiateView(&IssueCashView{
		IssueCash: &IssueCash{
			TokenType:     tokenType,
			Quantity:      quantity,
			Recipient:     recipient,
			RecipientNode: recipientNode,
			Message:       message,
			Signature:     signature,
			Transaction:   transaction,
		},
	}, ctx)
	if err != nil {
		logger.Errorf("error issuing: %s", err.Error())
		return "", err
	}
	txID, ok := res.(string)
	if !ok {
		return "", errors.New("cannot parse issue token response")
	}
	logger.Infof("issued %d %s to [%s] on [%s] with message [%s], signature [%s], and transaction [%s]. ID: [%s]", quantity, tokenType, recipient, recipientNode, message, signature, transaction, txID)
	return txID, nil
}

// INTERNAL

// IssueOnSignatureResponse contains the output information of a tx containing a signature
type IssueOnSignatureResponse struct {
	// EncodedSignMessage is the base64 encoded signing message
	EncodedSignMessage string
	// EncodedTx is the base64 encoded transaction
	EncodedTx string
}

// VIEW

// IssueCash contains the input information to issue a token
type IssueCash struct {
	// TokenType is the type of token to issue
	TokenType string
	// Quantity represent the number of units of a certain token type stored in the token
	Quantity uint64
	// Recipient is an identifier of the recipient identity
	Recipient string
	// RecipientNode is the identifier of the node of the recipient
	RecipientNode string
	// Message is the message that will be visible to the recipient and the auditor
	Message string
	// Signature represents a signature signed offline
	Signature string
	// Transaction is the transaction in raw bytes
	Transaction string
}

type IssueCashView struct {
	*IssueCash
}

type IssueOnSignatureView struct {
}

func (v *IssueOnSignatureView) Call(vctx view.Context) (interface{}, error) {
	ctx := vctx.Context()

	// At this point, the sender is ready to prepare the token transaction.
	// The sender creates an anonymous transaction (this means that the resulting Fabric transaction will be signed using idemix and the credentials are from the same network, for example).
	logger.DebugfContext(ctx, "Create anonymous transaction")
	tx, err := ttx.NewTransaction(
		vctx,
		nil,                                 // anonymous signer
		ttx.WithNoTransactionVerification(), // skip verification from committer
		ttx.WithNoCachingRequest(),          // skip db cache from orderer
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed creating anonymous transaction")
	}

	logger.DebugfContext(ctx, tx.TxID.String())
	logger.DebugfContext(ctx, tx.ID())
	logger.DebugfContext(ctx, tx.Network())
	logger.DebugfContext(ctx, tx.Channel())
	logger.DebugfContext(ctx, tx.Namespace())
	logger.DebugfContext(ctx, tx.Signer.String())
	logger.DebugfContext(ctx, fmt.Sprint(tx.Transient.IsEmpty()))

	// The issuer is ready to send transaction for client reference and message of the token request for offline signing.
	logger.Infof("sending transaction for references and message of token request for offline signing: [%s]", tx.ID())

	txRaw, err := tx.Bytes()
	if err != nil {
		return nil, err
	}
	requestRaw, err := tx.TokenRequest.MarshalToSign()
	if err != nil {
		return nil, err
	}

	// prepare the signing message
	encodedSignMessage := base64.StdEncoding.EncodeToString(requestRaw)
	// prepare the tx
	encodedTx := base64.StdEncoding.EncodeToString(txRaw)

	// preapre the issue on signature response
	issueOnSignatureRes := &IssueOnSignatureResponse{
		EncodedSignMessage: encodedSignMessage,
		EncodedTx:          encodedTx,
	}

	return issueOnSignatureRes, nil
}

func (v *IssueCashView) Call(vctx view.Context) (interface{}, error) {
	ctx := vctx.Context()
	wallet := ttx.MyIssuerWallet(vctx)
	if wallet == nil {
		return "", fmt.Errorf("issuer wallet not found")
	}

	rec := view.Identity(v.Recipient)
	eps := endpoint.GetService(vctx)
	err := eps.Bind(ctx, view.Identity(v.RecipientNode), rec)
	if err != nil {
		return "", fmt.Errorf("error binding %s to %s", v.Recipient, v.RecipientNode)
	}

	// As a first step operation, the issuer contacts the recipient's FSC node
	// to ask for the identity to use to assign ownership of the freshly created token.
	recipient, err := ttx.RequestRecipientIdentity(vctx, rec)
	if err != nil {
		return "", fmt.Errorf("failed getting recipient identity from %s: %w", v.RecipientNode, err)
	}

	var tx *ttx.Transaction
	// construct a tx from raw inputs
	if v.Transaction != "" {
		logger.Infof("parsing transaction from raw bytes to form a valid transaction")
		txRaw, err := base64.StdEncoding.DecodeString(v.Transaction)
		if err != nil {
			return "", fmt.Errorf("failed decoding transaction into raw bytes %s: %w", v.Transaction, err)
		}
		tx, err = ttx.NewTransactionFromBytes(
			vctx,
			txRaw,
		)
		if err != nil {
			return "", errors.Wrap(err, "failed creating transaction")
		}
	}

	// You can set any metadata you want. It is shared with the recipient and
	// auditor but not committed to the ledger. We used 'message' here to let
	// the user share messages that will be shown in the transaction history.
	if v.Message != "" {
		tx.SetApplicationMetadata("message", []byte(v.Message))
	}

	// The issuer adds a new issue operation to the transaction to issue
	// the amount to the recipient id recieved from the owner's node.
	if err = tx.Issue(
		wallet,
		recipient,
		token.Type(v.TokenType),
		v.Quantity,
	); err != nil {
		return "", errors.Wrap(err, "failed adding new issued token")
	}

	logger.DebugfContext(ctx, tx.TxID.String())
	logger.DebugfContext(ctx, tx.ID())
	logger.DebugfContext(ctx, tx.Network())
	logger.DebugfContext(ctx, tx.Channel())
	logger.DebugfContext(ctx, tx.Namespace())
	logger.DebugfContext(ctx, tx.Signer.String())
	logger.DebugfContext(ctx, fmt.Sprint(tx.Transient.IsEmpty()))

	// The issuer is ready to collect all the required signatures.
	// This includes the auditor (if provided) and the endorsers.
	logger.Infof("add the signatures to the token request of transaction: [%s]", tx.ID())

	if v.Signature != "" {
		signature, err := base64.StdEncoding.DecodeString(v.Signature)
		if err != nil {
			return "", fmt.Errorf("failed decoding signature into raw bytes %s: %w", v.Signature, err)
		}
		tx.TokenRequest.Actions.Signatures = append(tx.TokenRequest.Actions.Signatures, signature)
	}

	logger.DebugfContext(ctx, fmt.Sprint(tx.TokenRequest.Anchor))
	logger.DebugfContext(ctx, fmt.Sprint(tx.TokenRequest.Actions.Issues))
	logger.DebugfContext(ctx, fmt.Sprint(tx.TokenRequest.Actions.Signatures))
	logger.DebugfContext(ctx, fmt.Sprint(tx.TokenRequest.Metadata.Issues))
	logger.DebugfContext(ctx, fmt.Sprint(tx.TokenRequest.Metadata.Application))
	logger.DebugfContext(ctx, fmt.Sprint(tx.TokenRequest.TokenService.Network()))
	logger.DebugfContext(ctx, fmt.Sprint(tx.TokenRequest.TokenService.Channel()))
	logger.DebugfContext(ctx, fmt.Sprint(tx.TokenRequest.TokenService.Namespace()))

	// The issuer sends the transaction for ordering and finality.
	logger.Infof("submitting fabric transaction to orderer for final settlemement: [%s]", tx.ID())
	_, err = vctx.RunView(ttx.NewOrderingAndFinalityView(tx))
	if err != nil {
		return nil, errors.Wrap(err, "failed asking ordering")
	}

	logger.DebugfContext(ctx, fmt.Sprint(tx.Envelope.String()))

	return tx.ID(), nil
}
