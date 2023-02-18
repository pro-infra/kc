package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"

	certificatev1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func addUserCert(file string, days int, username string, groups ...string) {
	csr := newCSR(4096, username, groups...)
	cert := genK8SCertificate(file, csr, days)
	k := newKubeconfig(file)
	a := api.AuthInfo{
		ClientCertificateData: cert,
		ClientKeyData:         []byte(base64.StdEncoding.EncodeToString(csr.privatKey)),
	}
	k.AuthInfos[fmt.Sprintf("%s@%s", username, k.getClusterName())] = &a

	c := api.Context{
		Cluster:  k.getClusterName(),
		AuthInfo: fmt.Sprintf("%s@%s", username, k.getClusterName()),
	}
	k.Contexts[fmt.Sprintf("%s@%s", username, k.CurrentContext)] = &c
	err := clientcmd.WriteToFile(k.Config, file)
	if err != nil {
		panic(err)
	}
}

type csr struct {
	name      string
	groups    []string
	keyLength int
	pem       []byte
	privatKey []byte
}

func newCSR(length int, name string, groups ...string) csr {
	output := csr{keyLength: length, name: name, groups: groups}
	privateKey, err := rsa.GenerateKey(rand.Reader, length)
	if err != nil {
		panic(err)
	}
	output.privatKey = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	var csrTemplate = x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   name,
			Organization: groups,
		},
		SignatureAlgorithm: x509.SHA512WithRSA,
	}

	csrCertificate, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, privateKey)
	if err != nil {
		panic(err)
	}

	output.pem = pem.EncodeToMemory(&pem.Block{
		Type: "CERTIFICATE REQUEST", Bytes: csrCertificate,
	})

	return output
}

func genK8SCertificate(file string, certReq csr, days int) []byte {
	config, err := clientcmd.BuildConfigFromFlags("", file)
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	certificateClient := clientset.CertificatesV1()

	e := int32(days * 86400)

	certificateSigningRequest := certificatev1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: certReq.name,
		},
		Spec: certificatev1.CertificateSigningRequestSpec{
			Groups:            []string{"system:authenticated"},
			Request:           certReq.pem,
			SignerName:        "kubernetes.io/kube-apiserver-client",
			ExpirationSeconds: &e,
			Usages:            []certificatev1.KeyUsage{"client auth"},
		},
	}

	crl, err := certificateClient.CertificateSigningRequests().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	for _, i := range crl.Items {
		if i.Name == certReq.name {
			certificateClient.CertificateSigningRequests().Delete(context.TODO(), i.Name, metav1.DeleteOptions{})
		}
	}
	cr, err := certificateClient.CertificateSigningRequests().Create(context.TODO(), &certificateSigningRequest, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	cr.Status.Conditions = append(cr.Status.Conditions, certificatev1.CertificateSigningRequestCondition{
		Type:           certificatev1.RequestConditionType("Approved"),
		Status:         corev1.ConditionTrue,
		Reason:         "KubectlApprove",
		Message:        "This CSR was approved by kc certificate approve.",
		LastUpdateTime: metav1.Now(),
	})
	cr, err = certificateClient.CertificateSigningRequests().UpdateApproval(context.TODO(), cr.Name, cr, metav1.UpdateOptions{})
	if err != nil {
		panic(err)
	}
	w, err := certificateClient.CertificateSigningRequests().Watch(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", certReq.name),
	})
	if err != nil {
		panic(err)
	}
	for e := range w.ResultChan() {
		t := e.Object.(*certificatev1.CertificateSigningRequest)
		if e.Type == watch.Modified && string(t.Status.Certificate) != "" {
			w.Stop()
			return t.Status.Certificate
		}
	}
	return []byte{}
}
