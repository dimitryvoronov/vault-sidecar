/* This module provide tools for vault sidecar */
package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	vaultAPI "github.com/hashicorp/vault/api"
)

//Creates k8 secrets
func createK8Secret(name string, data map[string][]byte) {
	// keys k8s secret
	_, err := k8sClient().CoreV1().Secrets(vaultNamespace).Create(context.TODO(), &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind: "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: vaultNamespace,
		},
		Data: data,
	}, metav1.CreateOptions{})
	if errors.IsNotFound(err) {
		fmt.Printf("Secret %s not found\n", vaultInitKeys)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Printf("Error getting keys secret %v\n", statusError.ErrStatus.Message)
	} else if err != nil {
		panic(err.Error())
	} else {
		fmt.Printf("Secret %s has been created\n", name)
	}
}

//k8s clientset
func k8sClient() (k8sClient *kubernetes.Clientset) {
	configK8s, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(configK8s)
	if err != nil {
		panic(err.Error())

	}
	return clientset
}

//vault golang client
func vaultClient(token string, vaultAddr string) (vaultClient *vaultAPI.Client) {
	config := &vaultAPI.Config{
		Address: vaultAddr,
	}
	clientVault, err := vaultAPI.NewClient(config)
	if err != nil {
		log.Print(err)
	}
	clientVault.SetToken(token)
	return clientVault
}

//k8s secret token data
func k8sVaultSecret(k8sNamespace string) (data string, err error) {
	k8sKey, err := k8sClient().CoreV1().Secrets(vaultNamespace).Get(context.TODO(), vaultRootToken, metav1.GetOptions{})
	if err != nil {
		log.Fatal(err)
	}
	k8sKeyData, err := json.Marshal(k8sKey.Data)
	if err != nil {
		log.Fatal(err)
	}
	var k8sSecret K8sSecret
	err = json.Unmarshal(k8sKeyData, &k8sSecret)
	if err != nil {
		log.Println(err)
	}
	rootTokenDecrypt, err := base64.StdEncoding.DecodeString(k8sSecret.Token)
	if err != nil {
		log.Fatal(err)
	}
	var key = string(rootTokenDecrypt)

	return key, err
}
func k8sAnsibleSecret(k8sNamespace string) (data string, err error) {
	k8sKey, err := k8sClient().CoreV1().Secrets(k8sNamespace).Get(context.TODO(), k8sAnsibleKey, metav1.GetOptions{})
	if err != nil {
		log.Fatal(err)
	}
	k8sKeyData, err := json.Marshal(k8sKey.Data)
	if err != nil {
		log.Fatal(err)
	}
	var k8sSecret K8sSecret
	err = json.Unmarshal(k8sKeyData, &k8sSecret)
	if err != nil {
		log.Println(err)
	}
	rootTokenDecrypt, err := base64.StdEncoding.DecodeString(k8sSecret.KeySecret)
	if err != nil {
		log.Fatal(err)
	}
	var key = string(rootTokenDecrypt)

	return key, err
}
