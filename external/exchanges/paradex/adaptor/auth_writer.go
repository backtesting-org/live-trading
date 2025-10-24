package adaptor

import (
	"context"
	"fmt"
	"time"

	"github.com/trishtzy/go-paradex/auth"
	"github.com/trishtzy/go-paradex/client/authentication"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// autoAuthWriter injects the JWT into every request, refreshing if needed.
type autoAuthWriter struct {
	client *Client
	ctx    context.Context
}

func (a *autoAuthWriter) AuthenticateRequest(req runtime.ClientRequest, reg strfmt.Registry) error {
	// Check token without locking
	a.client.mu.RLock()
	tokenValid := a.client.jwtToken != "" && time.Now().Add(30*time.Second).Before(a.client.tokenExpiry)
	token := a.client.jwtToken
	a.client.mu.RUnlock()

	if tokenValid {
		return req.SetHeaderParam("Authorization", "Bearer "+token)
	}

	// Need to authenticate - but don't call Authenticate() to avoid double lock
	// Do the authentication inline
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()

	a.client.mu.Lock()
	defer a.client.mu.Unlock()

	// Double check after getting lock
	if a.client.jwtToken != "" && time.Now().Before(a.client.tokenExpiry) {
		return req.SetHeaderParam("Authorization", "Bearer "+a.client.jwtToken)
	}

	now := time.Now().Unix()
	timestamp := fmt.Sprintf("%d", now)
	expiration := fmt.Sprintf("%d", now+auth.DEFAULT_EXPIRY_IN_SECONDS)

	sig := auth.SignSNTypedData(auth.SignerParams{
		MessageType:       "auth",
		DexAccountAddress: a.client.dexAccountAddress,
		DexPrivateKey:     a.client.dexPrivateKey,
		SysConfig:         *a.client.systemConfig,
		Params: map[string]interface{}{
			"timestamp":  timestamp,
			"expiration": expiration,
		},
	})

	authParams := authentication.NewAuthParams().WithContext(ctx)
	authParams.SetPARADEXSTARKNETSIGNATURE(sig)
	authParams.SetPARADEXSTARKNETACCOUNT(a.client.dexAccountAddress)
	authParams.SetPARADEXTIMESTAMP(timestamp)
	authParams.SetPARADEXSIGNATUREEXPIRATION(&expiration)

	resp, err := a.client.api.Authentication.Auth(authParams)
	if err != nil {
		fmt.Println("Authentication failed:", err)
		return fmt.Errorf("authentication failed: %w", err)
	}

	a.client.jwtToken = resp.Payload.JwtToken
	a.client.tokenExpiry = time.Now().Add(time.Duration(auth.DEFAULT_EXPIRY_IN_SECONDS-30) * time.Second)

	return req.SetHeaderParam("Authorization", "Bearer "+a.client.jwtToken)
}

// AuthWriter returns an AuthInfoWriter for use in API calls.
func (c *Client) AuthWriter(ctx context.Context) runtime.ClientAuthInfoWriter {
	return &autoAuthWriter{client: c, ctx: ctx}
}
