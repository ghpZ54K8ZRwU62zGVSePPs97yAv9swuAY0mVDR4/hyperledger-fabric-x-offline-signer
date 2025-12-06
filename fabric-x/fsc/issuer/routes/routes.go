// SPDX-License-Identifier: MIT

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package routes

import (
	"context"
	"errors"

	"github.com/ghpZ54K8ZRwU62zGVSePPs97yAv9swuAY0mVDR4/hyperledger-fabric-x-offline-signer/fabric-x/fsc/issuer/service"
)

//go:generate go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=./oapi-server.yaml ../../swagger.yaml

type Server struct {
	fsc *service.FabricSmartClient
}

func NewServer(fsc *service.FabricSmartClient) Server {
	return Server{fsc: fsc}
}

// Issue tokens of any kind to an account on an offline signer
// (POST /issuer/issueonsignature)
func (s Server) IssueOnSignature(ctx context.Context, request IssueOnSignatureRequestObject) (IssueOnSignatureResponseObject, error) {
	if request.Body == nil {
		return nil, errors.New("no body")
	}
	res, err := s.fsc.IssueOnSignature(ctx)
	if err != nil {
		return nil, err
	}
	return IssueOnSignature200JSONResponse{IssueOnSignatureSuccessJSONResponse{
		Message:     "ok",
		SignMessage: res.EncodedSignMessage,
		Tx:          res.EncodedTx,
	}}, err
}

// Issue tokens of any kind to an account from an offline signature and a tx in raw
// (POST /issuer/issuetoken)
func (s Server) IssueToken(ctx context.Context, request IssueTokenRequestObject) (IssueTokenResponseObject, error) {
	if request.Body == nil {
		return nil, errors.New("no body")
	}
	var message string
	if request.Body.Message != nil {
		message = *request.Body.Message
	}
	var signature string
	if request.Body.Signature != nil {
		signature = *request.Body.Signature
	}
	var transaction string
	if request.Body.Transaction != nil {
		transaction = *request.Body.Transaction
	}
	res, err := s.fsc.IssueToken(ctx,
		request.Body.Amount.Code,
		request.Body.Amount.Value,
		request.Body.Counterparty.Account,
		request.Body.Counterparty.Node,
		message,
		signature,
		transaction,
	)
	if err != nil {
		return nil, err
	}
	return IssueToken200JSONResponse{IssueTokenSuccessJSONResponse{
		Message: "ok",
		Payload: res,
	}}, err
}

// Returns 200 if the service is healthy
// (GET /healthz)
func (s Server) Healthz(ctx context.Context, request HealthzRequestObject) (HealthzResponseObject, error) {
	return Healthz200JSONResponse{HealthSuccessJSONResponse{Message: "ok"}}, nil
}

// Returns 200 if the service is ready to accept calls
// (GET /readyz)
func (s Server) Readyz(ctx context.Context, request ReadyzRequestObject) (ReadyzResponseObject, error) {
	return Readyz200JSONResponse{HealthSuccessJSONResponse{Message: "ok"}}, nil
}
