package connector

import (
	"context"
	"fmt"
	"github.com/conductorone/baton-greenhouse/pkg/client"
	"github.com/conductorone/baton-greenhouse/pkg/models"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
)

type roleBuilder struct {
	Client *client.Client
}

const rolePermissionName = "assigned"

func (b *roleBuilder) ResourceType(_ context.Context) *v2.ResourceType {
	return roleResourceType
}

func (b *roleBuilder) List(_ context.Context, _ *v2.ResourceId, _ *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var roleResources []*v2.Resource

	siteAdminResource, err := createSiteAdminRoleResource()
	if err != nil {
		return nil, "", nil, err
	}

	roleResources = append(roleResources, siteAdminResource)

	return roleResources, "", nil, nil
}

func (b *roleBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var roleEntitlements []*v2.Entitlement

	assigmentOptions := []entitlement.EntitlementOption{
		entitlement.WithGrantableTo(userResourceType),
		entitlement.WithDescription(resource.Description),
		entitlement.WithDisplayName(resource.DisplayName),
	}

	roleEntitlements = append(roleEntitlements, entitlement.NewPermissionEntitlement(resource, rolePermissionName, assigmentOptions...))

	return roleEntitlements, "", nil, nil
}

func (b *roleBuilder) Grants(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var ret []*v2.Grant
	var err error

	users, err := b.listAllUsers(ctx)
	if err != nil {
		return nil, "", nil, err
	}

	for _, user := range users {
		if user.SiteAdmin {
			userResource, err := User2Resource(user, nil)
			if err != nil {
				return nil, "", nil, err
			}

			membershipGrant := grant.NewGrant(resource, rolePermissionName, userResource.Id)
			ret = append(ret, membershipGrant)
		}
	}

	return ret, "", nil, nil
}

func (b *roleBuilder) listAllUsers(ctx context.Context) ([]models.User, error) {
	var listedUsers []models.User
	var nextPageToken string
	for {
		users, _, nextToken, err := b.Client.ListUsers(ctx, nextPageToken)
		if err != nil {
			return nil, fmt.Errorf("error loading users cache, error: %w", err)
		}

		listedUsers = append(listedUsers, users...)

		if nextToken == "" {
			break
		}
		nextPageToken = nextToken
	}

	return listedUsers, nil
}

func createSiteAdminRoleResource() (*v2.Resource, error) {
	id := "site_admin"
	name := "Site Admin"
	profile := map[string]interface{}{
		"id":   id,
		"name": name,
	}

	roleTraits := []resourceSdk.RoleTraitOption{
		resourceSdk.WithRoleProfile(profile),
	}

	ret, err := resourceSdk.NewRoleResource(
		name,
		roleResourceType,
		id,
		roleTraits,
	)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func newRoleBuilder(c *client.Client) *roleBuilder {
	return &roleBuilder{
		Client: c,
	}
}
