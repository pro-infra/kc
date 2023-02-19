package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"

	certificatev1 "k8s.io/api/certificates/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/certificates/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type certificateAuthRequest struct {
	name              string
	groups            []string
	expirationSeconds int32
	keyLength         int
}

const defaultKeyLength = 4096

func newCertificateAuthRequest(name string, groups ...string) certificateAuthRequest {
	return certificateAuthRequest{
		name:              name,
		groups:            groups,
		keyLength:         defaultKeyLength,
		expirationSeconds: 86400,
	}
}

func (c *certificateAuthRequest) setKeyLength(k int) {
	c.keyLength = k
}

func (c *certificateAuthRequest) getCertificateSigningRequest() (csr certificateSigningRequest, err error) {

	privateKey, err := rsa.GenerateKey(rand.Reader, c.keyLength)
	if err != nil {
		return
	}
	if err != nil {
		return
	}
	csr.privatKey = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	var csrTemplate = x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: c.name,
		},
		SignatureAlgorithm: x509.SHA512WithRSA,
	}
	if len(c.groups) > 0 {
		csrTemplate.Subject.Organization = c.groups
	}

	csrCertificate, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, privateKey)
	if err != nil {
		return
	}

	csr.pem = pem.EncodeToMemory(&pem.Block{
		Type: "CERTIFICATE REQUEST", Bytes: csrCertificate,
	})

	return
}

type certificateSigningRequest struct {
	pem       []byte
	privatKey []byte
}

type certificateClient struct {
	client v1.CertificatesV1Interface
}

func newCertificateClient(file string) (cc certificateClient, err error) {

	config, err := clientcmd.BuildConfigFromFlags("", file)
	if err != nil {
		return
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return
	}
	c := clientset.CertificatesV1()
	cc.client = c
	return
}

func (c certificateClient) sendCertificateSigningRequest(name string, csr certificateSigningRequest, expirationSeconds int32) (cr *certificatev1.CertificateSigningRequest, err error) {
	certificateSigningRequest := certificatev1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: certificatev1.CertificateSigningRequestSpec{
			Groups:            []string{"system:authenticated"},
			Request:           csr.pem,
			SignerName:        "kubernetes.io/kube-apiserver-client",
			ExpirationSeconds: &expirationSeconds,
			Usages:            []certificatev1.KeyUsage{"client auth"},
		},
	}
	cr, err = c.client.CertificateSigningRequests().Create(context.TODO(), &certificateSigningRequest, metav1.CreateOptions{})
	return
}

func (c certificateClient) deleteCertificateSigningRequest(name string) error {
	return c.client.CertificateSigningRequests().Delete(context.TODO(), name, metav1.DeleteOptions{})
}

func (c certificateClient) approveAndSignCertificateSigningRequest(cr *certificatev1.CertificateSigningRequest) (certificate []byte, err error) {
	certificate = []byte{}
	cr.Status.Conditions = append(cr.Status.Conditions, certificatev1.CertificateSigningRequestCondition{
		Type:           certificatev1.RequestConditionType("Approved"),
		Status:         corev1.ConditionTrue,
		Reason:         "KubectlApprove",
		Message:        "This CSR was approved by kc certificate approve.",
		LastUpdateTime: metav1.Now(),
	})
	cr, err = c.client.CertificateSigningRequests().UpdateApproval(context.TODO(), cr.Name, cr, metav1.UpdateOptions{})
	if err != nil {
		return
	}
	w, err := c.client.CertificateSigningRequests().Watch(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", cr.Name),
	})
	if err != nil {
		return
	}

	for e := range w.ResultChan() {
		t := e.Object.(*certificatev1.CertificateSigningRequest)
		certificate = t.Status.Certificate
		if e.Type == watch.Modified && string(certificate) != "" {
			w.Stop()
			return
		}
	}

	return
}

func addUserCert(file string, days int, username string, groups ...string) {
	authRequest := newCertificateAuthRequest(username, groups...)
	csr, err := authRequest.getCertificateSigningRequest()
	if err != nil {
		panic(err)
	}

	cc, err := newCertificateClient(file)
	if err != nil {
		panic(err)
	}

	cr, err := cc.sendCertificateSigningRequest(username, csr, int32(days*86400))
	if errors.IsAlreadyExists(err) {
		if err = cc.deleteCertificateSigningRequest(username); err != nil {
			panic(err)
		}
		if cr, err = cc.sendCertificateSigningRequest(username, csr, int32(days*86400)); err != nil {
			panic(err)
		}
	}

	if err != nil {
		panic(err)
	}

	cert, err := cc.approveAndSignCertificateSigningRequest(cr)
	if err != nil {
		panic(err)
	}

	a := api.AuthInfo{
		ClientCertificateData: cert,
		ClientKeyData:         csr.privatKey,
	}

	k := newKubeconfig(file)
	k.AuthInfos[fmt.Sprintf("%s@%s", username, k.getClusterName())] = &a

	c := api.Context{
		Cluster:  k.getClusterName(),
		AuthInfo: fmt.Sprintf("%s@%s", username, k.getClusterName()),
	}
	k.Contexts[fmt.Sprintf("%s@%s", username, k.getClusterName())] = &c

	err = clientcmd.WriteToFile(k.Config, file)
	if err != nil {
		panic(err)
	}
}
