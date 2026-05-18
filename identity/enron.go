package identity

import (
	"github.com/grokify/commgraph/entity"
)

// SCIMUser represents a user in SCIM format (simplified for enron-people compatibility).
type SCIMUser struct {
	Active      bool
	DisplayName string
	Emails      []SCIMEmail
	ExternalID  string
	Groups      []SCIMGroup
	Locale      string
	Name        SCIMName
	NickName    string
	ProfileURL  string
	Timezone    string
	Title       string
	UserName    string
	UserType    string
}

// SCIMEmail represents an email in SCIM format.
type SCIMEmail struct {
	Value   string
	Type    string
	Primary bool
}

// SCIMName represents a name in SCIM format.
type SCIMName struct {
	Formatted       string
	GivenName       string
	MiddleName      string
	FamilyName      string
	HonorificSuffix string
}

// SCIMGroup represents a group in SCIM format.
type SCIMGroup struct {
	Display string
}

// LoadSCIMUsers loads SCIM users into the resolver.
func (r *SCIMResolver) LoadSCIMUsers(users []SCIMUser) {
	for _, user := range users {
		actor := SCIMUserToActor(user)
		r.LoadActor(actor)
	}
}

// SCIMUserToActor converts a SCIM user to an Actor.
func SCIMUserToActor(user SCIMUser) *entity.Actor {
	// Extract emails
	var emails []string
	var primaryEmail string
	for _, email := range user.Emails {
		emails = append(emails, email.Value)
		if email.Primary {
			primaryEmail = email.Value
		}
	}
	if primaryEmail == "" && len(emails) > 0 {
		primaryEmail = emails[0]
	}

	// Generate ID from username or primary email
	id := user.UserName
	if id == "" {
		id = primaryEmail
	}

	// Extract department from groups
	var department string
	if len(user.Groups) > 0 {
		department = user.Groups[0].Display
	}

	// Build display name
	displayName := user.DisplayName
	if displayName == "" {
		if user.Name.Formatted != "" {
			displayName = user.Name.Formatted
		} else if user.Name.GivenName != "" || user.Name.FamilyName != "" {
			displayName = user.Name.GivenName
			if user.Name.FamilyName != "" {
				if displayName != "" {
					displayName += " "
				}
				displayName += user.Name.FamilyName
			}
		}
	}

	return &entity.Actor{
		ID:           entity.ActorID(id),
		DisplayName:  displayName,
		Emails:       emails,
		PrimaryEmail: primaryEmail,
		ExternalID:   user.ExternalID,
		Internal:     true, // SCIM data is typically internal
		Department:   department,
		Title:        user.Title,
		Timezone:     user.Timezone,
		Metadata: map[string]string{
			"username":   user.UserName,
			"nickname":   user.NickName,
			"profile":    user.ProfileURL,
			"user_type":  user.UserType,
			"locale":     user.Locale,
			"given":      user.Name.GivenName,
			"family":     user.Name.FamilyName,
			"middle":     user.Name.MiddleName,
			"suffix":     user.Name.HonorificSuffix,
			"active":     boolToString(user.Active),
		},
	}
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
