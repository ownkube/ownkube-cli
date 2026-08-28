package client

import (
	"context"
	"fmt"

	"github.com/ownkube/okctl/internal/api"
)

// GetWallet calls GET /v1/wallet — the prepaid wallet balance summary.
func (c *Client) GetWallet(ctx context.Context) (*api.WalletSummaryResponse, error) {
	resp, err := c.inner.GetV1WalletWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromGetWallet(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// GetCredit calls GET /v1/credit — signup-credit status and wallet balance.
func (c *Client) GetCredit(ctx context.Context) (*api.CreditStatusResponse, error) {
	resp, err := c.inner.GetV1CreditWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromGetCredit(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// ClaimCredit calls POST /v1/credit/claim — grants the one-time signup credit.
// The server gates eligibility; this is a no-op grant when already claimed.
func (c *Client) ClaimCredit(ctx context.Context) (*api.ClaimCreditResponse, error) {
	resp, err := c.inner.PostV1CreditClaimWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromClaimCredit(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// GetSpendControls calls GET /v1/spend-controls.
func (c *Client) GetSpendControls(ctx context.Context) (*api.SpendControlsResponse, error) {
	resp, err := c.inner.GetV1SpendControlsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromGetSpendControls(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// UpdateSpendControls calls PUT /v1/spend-controls with the given patch.
func (c *Client) UpdateSpendControls(ctx context.Context, body api.UpdateSpendControlsBody) (*api.SpendControls, error) {
	resp, err := c.inner.PutV1SpendControlsWithResponse(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromUpdateSpendControls(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// CheckoutSubscription calls POST /v1/checkout/subscription and returns a
// browser checkout URL for the chosen plan.
func (c *Client) CheckoutSubscription(ctx context.Context, tier api.SubscriptionCheckoutBodyTier) (*api.CheckoutUrlResponse, error) {
	resp, err := c.inner.PostV1CheckoutSubscriptionWithResponse(ctx, api.SubscriptionCheckoutBody{Tier: tier})
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromCheckoutSubscription(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// CheckoutTopUp calls POST /v1/checkout/top-up and returns a browser checkout
// URL that adds amountUsd to the wallet.
func (c *Client) CheckoutTopUp(ctx context.Context, amountUsd float32) (*api.CheckoutUrlResponse, error) {
	resp, err := c.inner.PostV1CheckoutTopUpWithResponse(ctx, api.TopUpCheckoutBody{AmountUsd: amountUsd})
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromCheckoutTopUp(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

// Portal calls POST /v1/portal and returns the billing portal URL.
func (c *Client) Portal(ctx context.Context) (*api.CheckoutUrlResponse, error) {
	resp, err := c.inner.PostV1PortalWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	if err := checkError(resp.StatusCode(), errorsFromPortal(resp), resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, unexpectedStatus(resp.StatusCode(), resp.Body)
	}
	return resp.JSON200, nil
}

func errorsFromGetWallet(r *api.GetV1WalletResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromGetCredit(r *api.GetV1CreditResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromClaimCredit(r *api.PostV1CreditClaimResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromGetSpendControls(r *api.GetV1SpendControlsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromUpdateSpendControls(r *api.PutV1SpendControlsResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromCheckoutSubscription(r *api.PostV1CheckoutSubscriptionResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromCheckoutTopUp(r *api.PostV1CheckoutTopUpResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}

func errorsFromPortal(r *api.PostV1PortalResponse) []*api.ErrorResponse {
	return []*api.ErrorResponse{r.JSON400, r.JSON401, r.JSON403, r.JSON404, r.JSON409, r.JSON412, r.JSON500}
}
