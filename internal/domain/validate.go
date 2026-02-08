package domain

import "fmt"

func ValidateLocation(loc Location) error {
	if loc.Lat < -90 || loc.Lat > 90 {
		return fmt.Errorf("lat out of range")
	}
	if loc.Lng < -180 || loc.Lng > 180 {
		return fmt.Errorf("lng out of range")
	}
	return nil
}

func ValidateRole(role string) bool {
	switch role {
	case RoleAdmin, RoleEndUser, RoleDrone:
		return true
	default:
		return false
	}
}

