package connector

import (
	"context"
	"fmt"
	"strconv"

	"github.com/conductorone/baton-greenhouse/pkg/client"
	"github.com/conductorone/baton-greenhouse/pkg/models"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/entitlement"
	resourceSdk "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

type roleBuilder struct {
	client *client.GreenhouseClient
}

const (
	rolePermissionName = "assigned"

	// User Roles can be type 'interviewer' or 'job_admin'.
	roleTypeInterviewer = "interviewer"
	roleTypeJobAdmin    = "job_admin"

	// This is a constant used to map the Entitlement created for the Site Admins.
	roleTypeSiteAdmin = "site_admin"
)

func (b *roleBuilder) ResourceType(_ context.Context) *v2.ResourceType {
	return roleResourceType
}

func (b *roleBuilder) List(ctx context.Context, _ *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var roleResources []*v2.Resource
	var outAnnotations annotations.Annotations
	logger := ctxzap.Extract(ctx)
	userRoles, rateLimitData, nextPageURL, err := b.client.ListUserRoles(ctx, pToken.Token)
	if err != nil {
		if rateLimitData != nil {
			outAnnotations.WithRateLimiting(rateLimitData)
		}
		return nil, "", outAnnotations, err
	}

	for _, userRole := range userRoles {
		// Only the roles of type "Job Admin" will be created.
		if userRole.Type != roleTypeJobAdmin {
			continue
		}

		newRoleResource, err := createRoleResource(
			strconv.Itoa(userRole.ID),
			userRole.Name,
			userRole.Type,
		)
		if err != nil {
			logger.Debug(
				fmt.Sprintf("Role resource creation failed. UserRoleID: %d; UserRoleType: %s", userRole.ID, userRole.Type),
				zap.Error(err),
			)
			return nil, "", nil, err
		}
		roleResources = append(roleResources, newRoleResource)

		// Creates the entitlement for the future job permission.
		newRoleResource, err = createRoleResource(
			fmt.Sprintf("future-job:%d", userRole.ID),
			fmt.Sprintf("Future Job: %s", userRole.Name),
			userRole.Type,
		)
		if err != nil {
			logger.Debug(
				fmt.Sprintf("Role resource (future-job) creation failed. UserRoleID: %d; UserRoleType: %s", userRole.ID, userRole.Type),
				zap.Error(err),
			)
			return nil, "", nil, err
		}
		roleResources = append(roleResources, newRoleResource)
	}

	// Creates the Role Resource for the 'Site Admin'
	siteAdminResource, err := createRoleResource(
		"site_admin",
		"Site Admin",
		roleTypeSiteAdmin,
	)
	if err != nil {
		return nil, "", nil, err
	}

	roleResources = append(roleResources, siteAdminResource)

	outAnnotations.WithRateLimiting(rateLimitData)
	return roleResources, nextPageURL, outAnnotations, nil
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

// Grants function is implemented in the users.go file.
func (b *roleBuilder) Grants(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func (b *roleBuilder) Grant(_ context.Context, _ *v2.Resource, _ *v2.Entitlement) (annotations.Annotations, error) {
	return nil, nil
}

// Revoke removes the Site Admin role from the user by setting their permission level to "basic".
func (b *roleBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	id := grant.Principal.Id.Resource

	userID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("failed to convert user ID to int: %w", err)
	}

	err = b.client.RevokeUserSiteAdmin(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke site admin role: %w", err)
	}

	return nil, nil
}

func extractUniqueUserRolesIDs(permissionList interface{}) ([]int, error) {
	userRoleIDs := make(map[int]struct{})

	switch sliceData := permissionList.(type) {
	case []models.JobPermission:
		for _, jobPermission := range sliceData {
			userRoleIDs[jobPermission.UserRoleID] = struct{}{}
		}

	case []models.FutureJobPermission:
		for _, futureJobPermission := range sliceData {
			userRoleIDs[futureJobPermission.UserRoleID] = struct{}{}
		}

	default:
		return nil, fmt.Errorf("unsupported data type: expected []StructA or []StructB, got %T", permissionList)
	}

	var IDs []int
	for ID := range userRoleIDs {
		IDs = append(IDs, ID)
	}

	return IDs, nil
}

func createRoleResource(roleID, name, roleType string) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"id":   roleID,
		"name": name,
		"type": roleType,
	}

	roleTraits := []resourceSdk.RoleTraitOption{}

	return resourceSdk.NewRoleResource(
		name,
		roleResourceType,
		roleID,
		roleTraits,
		resourceSdk.WithResourceProfile(profile),
	)
}

func newRoleBuilder(c *client.GreenhouseClient) *roleBuilder {
	return &roleBuilder{
		client: c,
	}
}
