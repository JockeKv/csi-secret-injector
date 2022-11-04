package main

import (
	"bytes"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"csi-secret-injector/pkg/cert"
	"csi-secret-injector/pkg/kubeclient"
	"csi-secret-injector/pkg/mutate"
)

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello %q", html.EscapeString(r.URL.Path))
	log.Println("Got connection on /")
}

func handleMutate(w http.ResponseWriter, r *http.Request) {

	log.Println("Got connection on /mutate")

	// read the body / request
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err)
	}

	// mutate the request
	mutated, err := mutate.Mutate(body, false)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%s", err)
	}

	// and write it back
	w.WriteHeader(http.StatusOK)
	w.Write(mutated)
}

func main() {
	var err error
	// Create certificates
	certConfig := cert.CertConfig{
		Name:      "csi-secret-injector",
		Namespace: "util",
		Org:       "xcxc.dev",
	}
	ca, err := certConfig.GenerateCACert()
	if err != nil {
		log.Fatal("could not create CA")
	}

	err = kubeclient.UpdateWebhookCA("csi-secret-injector-webhook", ca.Bytes())
	if err != nil {
		log.Fatalf("could not add ca to webhook: %v", err)
	}

	cert, key, err := certConfig.GenerateServerCert()
	if err != nil {
		log.Fatal("could not create server cert")
	}
	err = os.MkdirAll("/ssl/", 0666)
	if err != nil {
		log.Panic(err)
	}
	err = WriteFile("/ssl/tls.crt", cert)
	if err != nil {
		log.Panic(err)
	}

	err = WriteFile("/ssl/tls.key", key)
	if err != nil {
		log.Panic(err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/mutate", handleMutate)

	s := &http.Server{
		Addr:           ":8443",
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1048576
	}

	log.Fatal(s.ListenAndServeTLS("/ssl/tls.crt", "/ssl/tls.key"))
}

// WriteFile writes data in the file at the given path
func WriteFile(filepath string, sCert *bytes.Buffer) error {
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(sCert.Bytes())
	if err != nil {
		return err
	}
	return nil
}
