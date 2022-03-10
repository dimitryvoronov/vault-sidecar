/* This module initializes vault instance */
package cmd

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//Runs main logic for vaultsidecar
func Execute() {
	log.Println("Starting the vault-init service...")

	i, err := strconv.Atoi(checkInterval)
	if err != nil {
		log.Fatalf("CHECK_INTERVAL is invalid: %s", err)
	}
	checkIntervalDuration := time.Duration(i) * time.Second

	httpClient = http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	signalCh := make(chan os.Signal)
	signal.Notify(signalCh,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGKILL,
	)
	stop := func() {
		log.Printf("Shutting down")
		os.Exit(0)
	}

	for {
		select {
		case <-signalCh:
			stop()
		default:
		}
		response, err := httpClient.Head(vaultAddr + "/v1/sys/health")

		if response != nil && response.Body != nil {
			response.Body.Close()
		}
		if err != nil {
			log.Println(err)
			time.Sleep(checkIntervalDuration)
			continue
		}

		//switch response
		switch response.StatusCode {
		case 200:
			log.Println("Vault is initialized and unsealed.")
			createPolicyObjects()
			createObjects()
		case 429:
			log.Println("Vault is unsealed and in standby mode.")
		case 501:
			log.Println("Vault is not initialized. Initializing and unsealing...")
			initialize()
			unseal()
		case 503:
			log.Println("Vault is sealed. Unsealing...")
			unseal()
		default:
			log.Printf("Vault is in an unknown state. Status code: %d", response.StatusCode)
		}

		log.Printf("Next check in %s", checkIntervalDuration)

		select {
		case <-signalCh:
			stop()
		case <-time.After(checkIntervalDuration):
		}
	}
}

//Vault initialization
func initialize() {
	initRequest := InitRequest{
		SecretShares:    3,
		SecretThreshold: 3,
	}
	//log.Printf("init request with shares and threshood is ", initRequest)
	initRequestData, err := json.Marshal(&initRequest)
	if err != nil {
		log.Println(err)
		return
	}

	//Check if k8s secrets exists and delete them since initialize is triggered.
	vaultRootTokenSecret, err := k8sClient().CoreV1().Secrets(vaultNamespace).Get(context.TODO(), vaultRootToken, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		fmt.Printf("Secret %s not found\n", vaultRootTokenSecret)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Printf("Error getting root token secret %v\n", statusError.ErrStatus.Message)
	} else if err != nil {
		panic(err.Error())
	} else {
		fmt.Printf("Secret %s has been found\n", vaultRootTokenSecret.Name)
	}
	vaultInitKeysSecret, err := k8sClient().CoreV1().Secrets(vaultNamespace).Get(context.TODO(), vaultInitKeys, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		fmt.Printf("Secret %s not found\n", vaultInitKeysSecret)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Printf("Error getting init keys secret %v\n", statusError.ErrStatus.Message)
	} else if err != nil {
		panic(err.Error())
	} else {
		fmt.Printf("Secret %s has been found\n", vaultInitKeysSecret.Name)
	}
	//check if k8s secrets for vault token and keys are present on EKS cluster
	if vaultRootTokenSecret.Name != "" || vaultInitKeysSecret.Name != "" {
		log.Printf("%s or %s secret exists, deleting the secret", vaultRootToken, vaultInitKeys)
		k8sSecrets = append(k8sSecrets, vaultRootToken, vaultInitKeys)
		for _, secret := range k8sSecrets {
			log.Printf("Removing the secrets %s\n", secret)
			_ = k8sClient().CoreV1().Secrets(vaultNamespace).Delete(context.TODO(), secret, metav1.DeleteOptions{})
			if errors.IsNotFound(err) {
				fmt.Printf("Secret %s not found\n", secret)
			} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
				fmt.Printf("Error getting init keys secret %v\n", statusError.ErrStatus.Message)
			} else if err != nil {
				panic(err.Error())
			} else {
				fmt.Printf("Secret %s has been found\n", secret)
			}
		}
	} else {
		log.Printf("%s or %s does not exists, creating ...", vaultRootToken, vaultInitKeys)
	}

	//main: initialization request for vault
	r := bytes.NewReader(initRequestData)
	request, err := http.NewRequest("PUT", vaultAddr+"/v1/sys/init", r)
	if err != nil {
		log.Println(err)
		return
	}
	response, err := httpClient.Do(request)
	if err != nil {
		log.Println(err)
		return
	}
	defer response.Body.Close()

	initRequestResponseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return
	}

	if response.StatusCode != 200 {
		log.Printf("init: non 200 status code: %d", response.StatusCode)
		return
	}

	var initResponse InitResponse

	if err := json.Unmarshal(initRequestResponseBody, &initResponse); err != nil {
		log.Println(err)
		return
	}

	//Create k8s secret for root token
	data_token := make(map[string][]byte)
	data_token["token"] = []byte(initResponse.RootToken)

	data_keys := make(map[string][]byte)
	for i, key := range initResponse.KeysBase64 {
		data_keys[(fmt.Sprintf("vault-key-%v", i))] = []byte(key)
		if err != nil {
			log.Println(err)
			return
		}
	}
	//token k8s secrets
	createK8Secret(vaultRootToken, data_token)
	createK8Secret(vaultInitKeys, data_keys)
	log.Println("Initialization of vault has been completed")
}
