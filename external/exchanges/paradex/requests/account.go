package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/trishtzy/go-paradex/client/account"
	"github.com/trishtzy/go-paradex/models"
)

func (s *Service) GetAccount(ctx context.Context) (*models.ResponsesAccountSummaryResponse, error) {
	params := account.NewGetAccountParams().WithContext(ctx)
	resp, err := s.client.API().Account.GetAccount(params, s.client.AuthWriter(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}

	return resp.Payload, nil
}

func (s *Service) GetAssetBalances(ctx context.Context) ([]*models.ResponsesBalanceResp, error) {
	params := account.NewGetBalanceParams().WithContext(ctx)
	resp, err := s.client.API().Account.GetBalance(params, s.client.AuthWriter(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get account balance: %w", err)
	}
	return resp.Payload.Results, nil
}

func (s *Service) GetProfile(ctx context.Context) (*models.ResponsesAccountProfileResp, error) {
	params := account.NewGetAccountProfileParams().WithContext(ctx)
	resp, err := s.client.API().Account.GetAccountProfile(params, s.client.AuthWriter(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	return resp.Payload, nil
}

func (s *Service) GetTradeHistory(ctx context.Context, market *string, limit *int64) (*models.ResponsesGetFillsResp, error) {
	params := account.NewGetFillsParams().WithContext(ctx)
	params.SetMarket(market)
	if limit != nil {
		params.SetPageSize(limit)
	}
	resp, err := s.client.API().Account.GetFills(params, s.client.AuthWriter(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get trade history: %w", err)
	}
	return resp.Payload, nil
}

func (s *Service) GetUserPositions(ctx context.Context) (*models.ResponsesGetPositionsResp, error) {
	params := account.NewGetPositionsParams().WithContext(ctx)
	resp, err := s.client.API().Account.GetPositions(params, s.client.AuthWriter(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get user orders: %w", err)
	}
	return resp.Payload, nil
}

func (s *Service) GetSubAccounts(ctx context.Context) (*models.ResponsesGetSubAccountsResponse, error) {
	params := account.NewGetSubAccountsParams().WithContext(ctx)
	resp, err := s.client.API().Account.GetSubAccounts(params, s.client.AuthWriter(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get sub-accounts: %w", err)
	}
	return resp.Payload, nil
}

// AccountInfoResponse is a custom response type to work around SDK bug
// The SDK has Kind as a nested struct, but API returns it as a plain string
type AccountInfoResponse struct {
	Account        string `json:"account,omitempty"`
	CreatedAt      int64  `json:"created_at,omitempty"`
	DerivationPath string `json:"derivation_path,omitempty"`
	IsolatedMarket string `json:"isolated_market,omitempty"`
	Kind           string `json:"kind,omitempty"` // Fixed: should be string, not struct
	ParentAccount  string `json:"parent_account,omitempty"`
	PublicKey      string `json:"public_key,omitempty"`
	Username       string `json:"username,omitempty"`
}

type GetAccountsInfoResponse struct {
	Results []*AccountInfoResponse `json:"results"`
}

func (s *Service) GetAccountInfo(ctx context.Context) (*GetAccountsInfoResponse, error) {
	// Make raw HTTP request to work around SDK bug with Kind field
	// The SDK expects Kind to be a nested struct, but API returns it as a string

	// Get the base URL from the client
	baseURL := s.client.GetBaseURL()

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/account/info", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication headers
	authHeaders := s.client.GetAuthHeaders()
	for key, value := range authHeaders {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result GetAccountsInfoResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}
