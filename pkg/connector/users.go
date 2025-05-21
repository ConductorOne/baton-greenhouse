package connector

import (
	"context"
	"fmt"
	"strconv"

	"github.com/conductorone/baton-greenhouse/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
)

type userBuilder struct {
	client *client.GreenhouseClient
}

func (b *userBuilder) ResourceType(_ context.Context) *v2.ResourceType {
	return userResourceType
}

func (b *userBuilder) List(ctx context.Context, _ *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var userResources []*v2.Resource
	var outAnnotations annotations.Annotations

	users, rateLimitData, next, err := b.client.ListUsers(ctx, pToken.Token)
	if err != nil {
		if rateLimitData != nil {
			outAnnotations.WithRateLimiting(rateLimitData)
		}
		return nil, "", outAnnotations, fmt.Errorf("cannot list users, error: %w", err)
	}

	for _, user := range users {
		userResource, err := parseIntoUserResource(user)
		if err != nil {
			return nil, "", nil, err
		}

		userResources = append(userResources, userResource)
	}

	outAnnotations.WithRateLimiting(rateLimitData)

	return userResources, next, outAnnotations, nil
}

// Entitlements always returns an empty slice for users.
func (b *userBuilder) Entitlements(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// Grants function will create the Grants for the Roles. This should upgrade performance and reduce sync time.
func (b *userBuilder) Grants(ctx context.Context, userResource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var roleGrants []*v2.Grant
	var outAnnotations annotations.Annotations
	userID := userResource.Id

	tokens, err := client.DeserializeTokens(pToken.Token)
	if err != nil {
		return nil, "", nil, err
	}

	user, rateLimitData, err := b.client.RetrieveUserData(ctx, userResource.Id.Resource)
	if err != nil {
		if rateLimitData != nil {
			outAnnotations.WithRateLimiting(rateLimitData)
		}
		return nil, "", outAnnotations, fmt.Errorf("cannot retrieve user: %w", err)
	}
	outAnnotations.WithRateLimiting(rateLimitData)

	// If the user is a Site Admin, it should have that Grant and skip the other ones.
	if user.SiteAdmin {
		roleResource := &v2.Resource{
			Id: &v2.ResourceId{
				ResourceType: roleResourceType.Id,
				Resource:     "site_admin",
			},
		}

		membershipGrant := grant.NewGrant(roleResource, rolePermissionName, userID)
		roleGrants = append(roleGrants, membershipGrant)
	} else {
		// All the Job Permissions of the user will be requested in order to create a grant for any role
		// for which the user has at least one Job with it.
		userJobPermissions, rateLimitData, err := b.client.GetJobPermissionsOfAUser(ctx, &tokens, user.ID)
		if err != nil {
			if rateLimitData != nil {
				outAnnotations.WithRateLimiting(rateLimitData)
			}
			return nil, "", outAnnotations, err
		}
		outAnnotations.WithRateLimiting(rateLimitData)

		uniqueUserRoleIDs, err := extractUniqueUserRolesIDs(userJobPermissions)
		if err != nil {
			return nil, "", nil, err
		}
		for _, userRoleID := range uniqueUserRoleIDs {
			roleResource := &v2.Resource{
				Id: &v2.ResourceId{
					ResourceType: roleResourceType.Id,
					Resource:     strconv.Itoa(userRoleID),
				},
			}

			membershipGrant := grant.NewGrant(roleResource, rolePermissionName, userID)
			roleGrants = append(roleGrants, membershipGrant)
		}

		// Retrieves the list of 'Future Job Permissions' assigned to the user.
		// These are Job Permissions that will be granted to the user when a job is created in a particular Department/Office combination.
		userFutureJobPermissions, rateLimitData, err := b.client.GetFutureJobPermissionsOfAUser(ctx, &tokens, user.ID)
		if err != nil {
			if rateLimitData != nil {
				outAnnotations.WithRateLimiting(rateLimitData)
			}
			return nil, "", outAnnotations, err
		}
		outAnnotations.WithRateLimiting(rateLimitData)

		uniqueUserRoleIDs, err = extractUniqueUserRolesIDs(userFutureJobPermissions)
		if err != nil {
			return nil, "", nil, err
		}
		for _, userRoleID := range uniqueUserRoleIDs {
			futureRoleUserID := fmt.Sprintf("future-job:%d", userRoleID)
			roleResource := &v2.Resource{
				Id: &v2.ResourceId{
					ResourceType: roleResourceType.Id,
					Resource:     futureRoleUserID,
				},
			}

			membershipGrant := grant.NewGrant(roleResource, rolePermissionName, userID)
			roleGrants = append(roleGrants, membershipGrant)
		}
	}

	var nextToken string
	if tokens.JobPermissionsToken == client.RequestCompleted && tokens.FutureJobPermissionsToken == client.RequestCompleted {
		nextToken = ""
	} else {
		nextToken, err = client.SerializeTokens(tokens)
		if err != nil {
			return nil, "", outAnnotations, err
		}
	}

	return roleGrants, nextToken, outAnnotations, nil
}

// CreateAccountCapabilityDetails returns the account provisioning capabilities of this connector.
// In this case, only account creation without password is supported.
func (b *userBuilder) CreateAccountCapabilityDetails(
	_ context.Context,
) (*v2.CredentialDetailsAccountProvisioning, annotations.Annotations, error) {
	return &v2.CredentialDetailsAccountProvisioning{
		SupportedCredentialOptions: []v2.CapabilityDetailCredentialOption{
			v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD,
		},
		PreferredCredentialOption: v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD,
	}, nil, nil
}

func (b *userBuilder) CreateAccount(
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

	user, err := b.client.GetAdminByEmail(ctx, b.client.GetOnBehalfOfEmail())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to resolve user ID: %w", err)
	}

	createdUser, err := b.client.CreateUserAccount(ctx, user.ID, email, firstName, lastName)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create user: %w", err)
	}

	res, err := resource.NewUserResource(
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

	return &v2.CreateAccountResponse_SuccessResult{Resource: res}, nil, nil, nil
}

func newUserBuilder(c *client.GreenhouseClient) *userBuilder {
	return &userBuilder{
		client: c,
	}
}
