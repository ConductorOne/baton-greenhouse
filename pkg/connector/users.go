package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-greenhouse/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
)

type userBuilder struct {
	client *client.Client
}

func newUserBuilder(c *client.Client) *userBuilder {
	return &userBuilder{
		client: c,
	}
}

func (o *userBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return userResourceType
}

func (o *userBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	list, rl, next, err := o.client.ListUsers(ctx, pToken.Token)
	if err != nil {
		return nil, "", nil, fmt.Errorf("cannot list users, error: %w", err)
	}
	users, err := Users2Resources(list, parentResourceID)
	if err != nil {
		return nil, "", nil, err
	}

	var anno annotations.Annotations
	anno.WithRateLimiting(rl)

	return users, next, anno, nil
}

func (o *userBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func (o *userBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// CreateAccountCapabilityDetails returns the account provisioning capabilities of this connector.
// In this case, only account creation without password is supported.
func (o *userBuilder) CreateAccountCapabilityDetails(
	ctx context.Context,
) (*v2.CredentialDetailsAccountProvisioning, annotations.Annotations, error) {
	return &v2.CredentialDetailsAccountProvisioning{
		SupportedCredentialOptions: []v2.CapabilityDetailCredentialOption{
			v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD,
		},
		PreferredCredentialOption: v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD,
	}, nil, nil
}

// CreateAccount creates a new user account in Greenhouse.
func (o *userBuilder) CreateAccount(
	ctx context.Context,
	accountInfo *v2.AccountInfo,
	_ *v2.CredentialOptions,
) (
	connectorbuilder.CreateAccountResponse,
	[]*v2.PlaintextData,
	annotations.Annotations,
	error,
) {
	profile := accountInfo.GetProfile().AsMap()

	email, ok := profile["email"].(string)
	if !ok {
		return nil, nil, nil, fmt.Errorf("missing or invalid 'email' in profile")
	}
	firstName, ok := profile["first_name"].(string)
	if !ok {
		return nil, nil, nil, fmt.Errorf("missing or invalid 'first_name' in profile")
	}
	lastName, ok := profile["last_name"].(string)
	if !ok {
		return nil, nil, nil, fmt.Errorf("missing or invalid 'last_name' in profile")
	}

	user, err := o.client.GetAdminByEmail(ctx, o.client.GetOnBehalfOfEmail())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to resolve user ID: %w", err)
	}

	createdUser, err := o.client.CreateUserAccount(ctx, user.ID, email, firstName, lastName)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create user: %w", err)
	}

	resource, err := resource.NewUserResource(
		firstName+" "+lastName,
		userResourceType,
		createdUser.ID,
		[]resource.UserTraitOption{
			resource.WithEmail(email, true),
			resource.WithUserProfile(map[string]interface{}{
				"email":      email,
				"first_name": firstName,
				"last_name":  lastName,
				"id":         createdUser.ID,
			}),
		},
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to build resource: %w", err)
	}

	return &v2.CreateAccountResponse_SuccessResult{Resource: resource}, nil, nil, nil
}
