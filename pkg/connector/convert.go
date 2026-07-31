package connector

import (
	"fmt"

	"github.com/conductorone/baton-greenhouse/pkg/models"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
)

func parseIntoUserResource(user models.User) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"first_name":  user.FirstName,
		"last_name":   user.LastName,
		"employee_id": user.EmployeeID,
		"is_admin":    user.SiteAdmin,
	}

	options := []resource.UserTraitOption{
		resource.WithEmployeeID(user.EmployeeID),
		resource.WithEmail(user.PrimaryEmailAddress, true),
	}

	resourceOptions := []resource.ResourceOption{
		resource.WithResourceProfile(profile),
	}

	if user.Disabled {
		resourceOptions = append(resourceOptions, resource.WithResourceStatus(v2.Status_RESOURCE_STATUS_DISABLED, ""))
	} else {
		resourceOptions = append(resourceOptions, resource.WithResourceStatus(v2.Status_RESOURCE_STATUS_ENABLED, ""))
	}

	userResource, err := resource.NewUserResource(
		fmt.Sprintf("%s %s", user.FirstName, user.LastName),
		userResourceType,
		user.ID,
		options,
		resourceOptions...,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot make user resource from user «%s %s» (id «%d»)", user.FirstName, user.LastName, user.ID)
	}

	return userResource, nil
}
