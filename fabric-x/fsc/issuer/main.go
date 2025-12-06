// SPDX-License-Identifier: MIT

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ghpZ54K8ZRwU62zGVSePPs97yAv9swuAY0mVDR4/hyperledger-fabric-x-offline-signer/fabric-x/fsc/common"
	"github.com/ghpZ54K8ZRwU62zGVSePPs97yAv9swuAY0mVDR4/hyperledger-fabric-x-offline-signer/fabric-x/fsc/issuer/routes"
	"github.com/ghpZ54K8ZRwU62zGVSePPs97yAv9swuAY0mVDR4/hyperledger-fabric-x-offline-signer/fabric-x/fsc/issuer/service"
	viewregistry "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/view"

	// TODO: don't use integration views
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/views"
)

func main() {
	// Flags
	cwd, _ := os.Getwd()
	pth := flag.String("conf", cwd, "the directory that contains the core.yaml configuration file")
	datadir := flag.String("datadir", cwd, "the directory that contains the database")
	port := flag.String("port", "9000", "the API port for the application")
	flag.Parse()

	// Fabric smart client
	fsc, err := common.StartFSC(*pth, *datadir)
	if err != nil {
		log.Fatal(err)
	}

	// Register views and responders (communication with other FSC nodes)
	reg := viewregistry.GetRegistry(fsc)
	reg.RegisterFactory("issue", &views.IssueCashViewFactory{})
	reg.RegisterResponder(&views.IssuerRedeemAcceptView{}, &views.RedeemView{})

	// Simple web server
	sh := routes.NewStrictHandler(routes.NewServer(service.NewFSC(fsc)), []routes.StrictMiddlewareFunc{})
	h := common.WithAnyCORS(routes.HandlerFromMux(sh, http.NewServeMux()))
	s := &http.Server{
		Handler: h,
		Addr:    net.JoinHostPort("0.0.0.0", *port),
	}
	go s.ListenAndServe()

	// Stop
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*10)
	defer cancel()
	s.Shutdown(ctx)
	fsc.Stop()
}
