/* This module creates objects on vault instance */
package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	vaultAPI "github.com/hashicorp/vault/api"
	avgo "github.com/sosedoff/ansible-vault-go"
)

//Creates Vault policies
func createPolicyObjects() {
	ts := time.Now()
	wg := new(sync.WaitGroup)
	e := filepath.Walk("/tmp/vault", func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && filepath.Ext(info.Name()) == ".hcl" {
			wg.Add(1)
			pn := info.Name()[0 : len(info.Name())-len(filepath.Ext(info.Name()))]
			go applyConfig(path, pn, wg)
		}
		return nil
	})
	wg.Wait()
	if e != nil {
		fmt.Print(e)
	}
	tf := time.Now()
	fmt.Println(tf.Sub(ts))
}

//Apply function for policy creation
func applyConfig(filePath, policyName string, wg *sync.WaitGroup) {
	defer wg.Done()
	vaultAnsibleKey, err := k8sAnsibleSecret(vaultNamespace)
	if err != nil {
		fmt.Print(err)
	}
	vaultAnsibleKey = strings.TrimRight(vaultAnsibleKey, "\n")
	token, err := k8sVaultSecret(vaultNamespace)
	if err != nil {
		fmt.Print(err)
	}

	decryptedFile, err := avgo.DecryptFile(filePath, vaultAnsibleKey)
	if err != nil {
		fmt.Print(err)
	}
	vaultClient(token, vaultAddr).Sys().PutPolicy(policyName, decryptedFile)
}

//Creates vault objects
func createObjects() {
	token, err := k8sVaultSecret(vaultNamespace)
	if err != nil {
		fmt.Print(err)
	}
	vaultAnsibleKey, err := k8sAnsibleSecret(vaultNamespace)
	if err != nil {
		fmt.Print(err)
	}
	vaultAnsibleKey = strings.TrimRight(vaultAnsibleKey, "\n")

	//Vault secret engines list (with already existing)
	engines := []string{
		"cubbyhole/",
		"identity/",
		"sys/",
		"environment/",
		"cluster/",
		"infrastructure/",
		"workflow/",
	}
	for i := 0; i < 1; i++ {
		listMounts, _ := vaultClient(token, vaultAddr).Sys().ListMounts()
		for _, engine := range engines {
			found := false
			for mount, _ := range listMounts {
				mount = string(mount)
				if mount == engine {
					found = true
					log.Printf("Secret engine %s exists\n", engine)
					break
				}
			}
			if !found {
				log.Printf("Secret engine %s not found, creating...\n", engine)
				err := vaultClient(token, vaultAddr).Sys().Mount(engine, &vaultAPI.MountInput{
					Type:        "kv",
					Description: fmt.Sprintf("KV %s secrets engine", engine),
					Config: vaultAPI.MountConfigInput{
						MaxLeaseTTL: "10800",
					},
				})
				if err != nil {
					fmt.Print(err)
				}
			}
		}
	}

	//Creating auth methods
	listAuthMethods := []string{
		"jwt/",
		"token/",
	}
	for i := 0; i < 1; i++ {
		auth, err := vaultClient(token, vaultAddr).Sys().ListAuth()
		if err != nil {
			fmt.Print(err)
		}
		for _, authMethod := range listAuthMethods {
			found := false
			for auth, _ := range auth {
				//auth := string(auth)
				if auth == authMethod {
					found = true
					fmt.Printf("Auth method %s exists\n", auth)
					break
				}
			}
			if !found {
				log.Printf("Auth method %v not exists, creating...\n", authMethod)
				err = vaultClient(token, vaultAddr).Sys().EnableAuthWithOptions(authMethod, &vaultAPI.EnableAuthOptions{
					Type: "jwt",
					Config: vaultAPI.MountConfigInput{
						DefaultLeaseTTL: "300",
						MaxLeaseTTL:     "600",
						TokenType:       "default-service",
					},
				})
				if err != nil {
					fmt.Print(err)
				}
			}
		}
	}
	log.Print("Creating a userpass auth method")
	err = vaultClient(token, vaultAddr).Sys().EnableAuthWithOptions("environment-userpass-pipeline", &vaultAPI.EnableAuthOptions{
		Type: "userpass",
		Config: vaultAPI.MountConfigInput{
			DefaultLeaseTTL: "2764800",
			MaxLeaseTTL:     "2764800",
			TokenType:       "default-service",
		},
	})
	if err != nil {
		log.Print(err)
	}

	//Creating Secrets
	walkDir := func(path string, info os.FileInfo, err error) error {
		if matches, err := filepath.Match(ocSecrets, info.Name()); matches && err == nil {

			decryptedFile, err := avgo.DecryptFile((path), vaultAnsibleKey)
			if err != nil {
				return err
			}
			var data map[string]interface{}
			err = json.Unmarshal([]byte(decryptedFile), &data)
			if err != nil {
				return err
			}
			// define cluster name based on filepath
			cluster := strings.Split(path, "/")
			_, err = vaultClient(token, vaultAddr).Logical().Write(fmt.Sprintf("environment/openshift/%s/%s", cluster[8], info.Name()), data)
			if err != nil {
				fmt.Print(err)
			}
		} else if matches && err != nil {
			log.Printf("Error, secret was not created, %s", err)
			return nil
		}
		return nil
	}
	//define walk dir path's
	if clusterOne != "" && clusterTwo != "" && clusterThree != "" {
		filepath.Walk(clusterOne, walkDir)
		filepath.Walk(clusterTwo, walkDir)
		filepath.Walk(clusterThree, walkDir)
	}

	//Creating additional keys
	vaultKeysPath = (append(vaultKeysPath, workFlowGard, infraKeys, awsSecrets, clusterAws, clusterOpenshift))
	for _, vPath := range vaultKeysPath {
		err = filepath.Walk(fmt.Sprint(basePath+vPath),
			func(path string, info os.FileInfo, err error) error {
				if info.IsDir() {
				} else {
					decryptedFile, err := avgo.DecryptFile((path), vaultAnsibleKey)
					if err != nil {
						fmt.Print(err)
					}
					var data map[string]interface{}
					err = json.Unmarshal([]byte(decryptedFile), &data)
					if err != nil {
						return err
					}
					_, err = vaultClient(token, vaultAddr).Logical().Write(fmt.Sprint(vPath+info.Name()), data)
					if err != nil {
						fmt.Print(err)
					}
				}
				return err
			})
		if err != nil {
			fmt.Print(err)
		}
	}
}
