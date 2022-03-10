/* This module does unseal of vault instance */

package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//Unseal vault
func unseal() {
	k8svaultInitKey, err := k8sClient().CoreV1().Secrets(vaultNamespace).Get(context.TODO(), vaultInitKeys, metav1.GetOptions{})
	if err != nil {
		log.Print(err)
	}
	keyData, err := json.Marshal(k8svaultInitKey.Data)
	if err != nil {
		log.Fatal(err)
	}

	var unsealResponse VaultKeys
	if err := json.Unmarshal(keyData, &unsealResponse); err != nil {
		log.Println(err)
		return
	}
	var unsealString []string
	unsealString = append(unsealString, unsealResponse.KeyOne, unsealResponse.KeyTwo, unsealResponse.KeyThree)
	log.Print("Starting unsealing of vault \n")

	for _, key := range unsealString {
		key_base64, err := base64.StdEncoding.DecodeString(key)
		if err != nil {
			fmt.Print(err)
		}
		key := string(key_base64)
		done, err := unsealOne(key)
		if done {
			return
		}
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func unsealOne(key string) (bool, error) {
	unsealRequest := UnsealRequest{
		Key: key,
	}

	unsealRequestData, err := json.Marshal(&unsealRequest)
	if err != nil {
		return false, err
	}

	r := bytes.NewReader(unsealRequestData)
	request, err := http.NewRequest(http.MethodPut, vaultAddr+"/v1/sys/unseal", r)
	if err != nil {
		return false, err
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return false, err
	}

	defer response.Body.Close()

	if response.StatusCode != 200 {
		return false, fmt.Errorf("unseal: non-200 status code: %d", response.StatusCode)
	}

	unsealRequestResponseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return false, err
	}

	var unsealResponse UnsealResponse
	if err := json.Unmarshal(unsealRequestResponseBody, &unsealResponse); err != nil {
		return false, err
	}

	if !unsealResponse.Sealed {
		return true, nil
	}
	return false, nil
}
