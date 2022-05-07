package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ackersonde/bender-slackbot/structures"
	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/approle"
)

var vaultClient *vault.Client

func initVaultClient() {
	if vaultClient != nil && vaultClient.Token() != "" {
		return
	}

	go renewToken()
	loginAttempts := 0
	for loginAttempts < 10 {
		loginAttempts += 1
		if vaultClient == nil || vaultClient.Token() == "" {
			time.Sleep(100 * time.Millisecond)
		} else {
			break
		}
	}
}

func readTOTPCodeForKey(totpEngineName string, keyName string) string {
	initVaultClient()

	response := ""
	code, err := vaultClient.Logical().Read(totpEngineName + "/code/" + keyName)

	if err != nil {
		if strings.Contains(err.Error(), "unknown key: "+keyName) {
			allKeys := listTOTPKeysForEngine(totpEngineName)
			// search thru remoteResults and attempt to do a case-insensitive match on keyname
			for _, key := range allKeys {
				lowerCaseKey := strings.ToLower(key)
				lowerCaseKeyName := strings.ToLower(keyName)

				if strings.Contains(lowerCaseKey, lowerCaseKeyName) {
					response += key + ": " + readTOTPCodeForKey(totpEngineName, key) + "\n"
				}
			}
			if response == "" {
				response = fmt.Sprintf("Unable to find keyName %s. Try looking at all keys w/ `vfa`\n", keyName)
			} else {
				response = fmt.Sprintf("Found the following keys:\n%s\n", response)
			}
		} else {
			response = fmt.Sprintf("unexpected error: %s\n", err.Error())
		}
	} else {
		response = "*" + code.Data["code"].(string) + "*"
	}

	return response
}

func putTOTPKeyForEngine(totpEngineName string, keyName string, secret string) string {
	initVaultClient()
	response := ""

	payload := map[string]interface{}{
		"key": secret,
	}
	_, err := vaultClient.Logical().Write(totpEngineName+"/keys/"+keyName, payload)

	if err != nil {
		response = fmt.Sprintf("Unable to add %s: %s", keyName, err.Error())
	} else {
		response = readTOTPCodeForKey(totpEngineName, keyName)
	}

	return response
}

func updateTOTPRoleCIDRs(roleName string, CIDRs string) string {
	initVaultClient()
	response := ""

	parts := strings.Split(CIDRs, ",")
	finalList := ""
	for _, part := range parts {
		finalList += part + ","
	}
	finalList += "127.0.0.1"

	payload := map[string]interface{}{
		"token_bound_cidrs": finalList,
	}
	_, err := vaultClient.Logical().Write("auth/approle/role/"+roleName, payload)
	if err != nil {
		response = fmt.Sprintf("Unable to update %s's CIDRs to %s: %s", roleName, finalList, err.Error())
	} else {
		response = "Updated TOTP Role `" + roleName + "` CIDRS to: " + CIDRs
	}
	return response
}

func listTOTPKeysForEngine(totpEngineName string) []string {
	initVaultClient()
	var response []string
	keys, err := vaultClient.Logical().List(totpEngineName + "/keys")

	if err != nil {
		response = append(response, err.Error())
	} else {
		for _, key := range keys.Data["keys"].([]interface{}) {
			response = append(response, key.(string))
		}
	}

	return response
}

func loginToVault() (*vault.Secret, error) {
	config := vault.DefaultConfig() // modify for more granular configuration

	vaultClient, _ = vault.NewClient(config)
	vaultClient.SetAddress(os.Getenv("VAULT_ADDR"))

	// A combination of a Role ID and Secret ID is required to log in to Vault
	// with an AppRole.
	// First, let's get the role ID given to us by our Vault administrator.
	roleID := os.Getenv("VAULT_APPROLE_ROLE_ID")
	if roleID == "" {
		Logger.Printf("no role ID was provided in VAULT_APPROLE_ROLE_ID env var")
	}

	// The Secret ID is a value that needs to be protected, so instead of the
	// app having knowledge of the secret ID directly, we have a trusted orchestrator (https://learn.hashicorp.com/tutorials/vault/secure-introduction?in=vault/app-integration#trusted-orchestrator)
	// give the app access to a short-lived response-wrapping token (https://www.vaultproject.io/docs/concepts/response-wrapping).
	// Read more at: https://learn.hashicorp.com/tutorials/vault/approle-best-practices?in=vault/auth-methods#secretid-delivery-best-practices
	secretID := &auth.SecretID{FromString: os.Getenv("VAULT_APPROLE_SECRET_ID")}

	appRoleAuth, err := auth.NewAppRoleAuth(roleID, secretID)
	if err != nil {
		Logger.Println(fmt.Errorf("unable to initialize AppRole auth method: %w", err))
	} else {
		authInfo, err := vaultClient.Auth().Login(context.Background(), appRoleAuth)
		if err != nil {
			Logger.Println(fmt.Errorf("%w", err))
			if strings.Contains(err.Error(), "Vault is sealed") {
				cmd := "/home/ubuntu/vault/unseal_vault.sh"
				remoteResult := executeRemoteCmd(cmd, structures.PI4RemoteConnectConfig)

				if remoteResult.Stdout == "" && remoteResult.Stderr != "" {
					Logger.Println(remoteResult.Stderr)
					err = errors.New(remoteResult.Stderr)
				} else {
					authInfo, err = vaultClient.Auth().Login(context.Background(), appRoleAuth)
					if authInfo != nil {
						err = errors.New("Successfully unsealed Vault")
					}
				}
			}
		}

		if authInfo == nil {
			Logger.Printf("no auth info was returned after login")
		}

		return authInfo, err
	}

	return nil, err
}

// Once you've set the token for your Vault client, you will need to
// periodically renew its lease.
func renewToken() {
	for {
		vaultLoginResp, err := loginToVault()
		if err != nil {
			Logger.Printf("unable to authenticate to Vault: %v\n", err)
			break
		} else {
			tokenErr := manageTokenLifecycle(vaultLoginResp)
			if tokenErr != nil {
				Logger.Printf("unable to start managing token lifecycle: %v\n", tokenErr)
				break
			}
		}
	}
}

// Starts token lifecycle management. Returns only fatal errors as errors,
// otherwise returns nil so we can attempt login again.
func manageTokenLifecycle(token *vault.Secret) error {
	renew := token.Auth.Renewable // You may notice a different top-level field called Renewable. That one is used for dynamic secrets renewal, not token renewal.
	if !renew {
		Logger.Printf("Token is not configured to be renewable. Re-attempting login.")
		return nil
	}

	watcher, err := vaultClient.NewLifetimeWatcher(&vault.LifetimeWatcherInput{
		Secret:    token,
		Increment: 3600, // Learn more about this optional value in https://www.vaultproject.io/docs/concepts/lease#lease-durations-and-renewal
	})
	if err != nil {
		return fmt.Errorf("unable to initialize new lifetime watcher for renewing auth token: %w", err)
	}

	go watcher.Start()
	defer watcher.Stop()

	for {
		select {
		// `DoneCh` will return if renewal fails, or if the remaining lease
		// duration is under a built-in threshold and either renewing is not
		// extending it or renewing is disabled. In any case, the caller
		// needs to attempt to log in again.
		case err := <-watcher.DoneCh():
			if err != nil {
				Logger.Printf("Failed to renew token: %v. Re-attempting login.", err)
				return nil
			}
			// This occurs once the token has reached max TTL.
			Logger.Printf("Token can no longer be renewed. Re-attempting login.")
			return nil

		// Successfully completed renewal
		case renewal := <-watcher.RenewCh():
			Logger.Printf("Successfully renewed: %#v", renewal)
		}
	}
}
