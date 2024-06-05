/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

const callback = "/auth/oidc/callback"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "examples",
	Short: "A brief description of your application",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if baseurl == "" {
			return os.ErrInvalid
		}

		ctx := context.Background()

		state, err := randString(16)
		if err != nil {
			log.Fatal("cannot create state")
			return
		}
		nonce, err := randString(16)
		if err != nil {
			log.Fatal("cannot create nonce")
			return
		}

		// open a listener, wait for token
		provider, err := oidc.NewProvider(ctx, baseurl)
		if err != nil {
			log.Fatal(err)
		}
		oidcConfig := &oidc.Config{
			ClientID: clientID,
		}
		verifier := provider.Verifier(oidcConfig)
		oauth2Verifier := oauth2.GenerateVerifier()

		config := oauth2.Config{
			ClientID:    clientID,
			Endpoint:    provider.Endpoint(),
			RedirectURL: fmt.Sprintf("http://127.0.0.1:%d%s", port, callback),
			Scopes:      []string{oidc.ScopeOpenID, "profile", "email"},
		}

		http.HandleFunc(callback, func(w http.ResponseWriter, r *http.Request) {
			oauth2Token, err := config.Exchange(ctx,
				r.URL.Query().Get("code"),
				oauth2.VerifierOption(oauth2Verifier),
			)
			if err != nil {
				http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
				return
			}
			rawIDToken, ok := oauth2Token.Extra("id_token").(string)
			if !ok {
				http.Error(w, "No id_token field in oauth2 token.", http.StatusInternalServerError)
				return
			}
			idToken, err := verifier.Verify(ctx, rawIDToken)
			if err != nil {
				http.Error(w, "Failed to verify ID Token: "+err.Error(), http.StatusInternalServerError)
				return
			}

			if idToken.Nonce != nonce {
				http.Error(w, "nonce does not match", http.StatusUnauthorized)
				return
			}

			oauth2Token.AccessToken = "*REDACTED*"

			resp := struct {
				OAuth2Token   *oauth2.Token
				IDTokenClaims *json.RawMessage // ID Token payload is just JSON.
			}{oauth2Token, new(json.RawMessage)}

			if err := idToken.Claims(&resp.IDTokenClaims); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			data, err := json.MarshalIndent(resp, "", "    ")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Write(data)
		})

		// build openid auth url and call browser
		authurl := config.AuthCodeURL(state,
			oidc.Nonce(nonce),
			oauth2.S256ChallengeOption(oauth2Verifier),
			oauth2.ApprovalForce,
			oauth2.SetAuthURLParam("prompt", "select_account"),
		)
		browser.OpenURL(authurl)

		log.Printf("listening on http://%s:%d/", "127.0.0.1", port)
		log.Printf("check nonce: %s", nonce)
		log.Fatal(http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", port), nil))

		return
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var baseurl, clientID string
var port int

func init() {
	rootCmd.Flags().StringVarP(&baseurl, "url", "u", "https://peter.itrslab.com/auth/realms/obcerv", "open auth url in browser")
	rootCmd.Flags().StringVarP(&clientID, "client", "c", "geneos-ui", "client-id")
	rootCmd.Flags().IntVarP(&port, "port", "P", 7070, "open auth url in browser")
}

func randString(nByte int) (string, error) {
	b := make([]byte, nByte)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
