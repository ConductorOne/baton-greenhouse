package connector

import (
	"fmt"

	"github.com/conductorone/baton-greenhouse/pkg/models"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
)

func User2Resource(u models.User, p *v2.ResourceId) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"first_name": u.Name,
		"last_name":  u.LastName,
		"is_admin":   u.SiteAdmin,
	}

	options := []resource.UserTraitOption{
		resource.WithUserProfile(profile),
		resource.WithEmail(u.PrimaryEmailAddress, true),
	}
	if u.Disabled {
		options = append(options, resource.WithStatus(v2.UserTrait_Status_STATUS_DISABLED))
	} else {
		options = append(options, resource.WithStatus(v2.UserTrait_Status_STATUS_ENABLED))
	}

	user, err := resource.NewUserResource(
		fmt.Sprintf("%s %s", u.Name, u.LastName),
		userResourceType,
		u.ID,
		options,
		resource.WithParentResourceID(p),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot make user resource from user «%s %s» (id «%d»)", u.Name, u.LastName, u.ID)
	}

	return user, nil
}

func Users2Resources(us []models.User, p *v2.ResourceId) ([]*v2.Resource, error) {
	var users []*v2.Resource

	for _, u := range us {
		user, err := User2Resource(u, p)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func Role2Resource(r models.Role, p *v2.ResourceId) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"Name": r.Name,
		"Type": r.Type,
	}

	options := []resource.RoleTraitOption{
		resource.WithRoleProfile(profile),
	}

	role, err := resource.NewRoleResource(r.Name, roleResourceType, r.ID, options)
	if err != nil {
		return nil, fmt.Errorf("cannot make role resource for «%s» (ID %d)", r.Name, r.ID)
	}

	return role, nil
}

func Roles2Resources(rs []models.Role, p *v2.ResourceId) ([]*v2.Resource, error) {
	var roles []*v2.Resource

	for _, r := range rs {
		role, err := Role2Resource(r, p)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	return roles, nil
}
