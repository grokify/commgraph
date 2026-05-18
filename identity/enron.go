package identity

import (
	"strings"

	"github.com/enrondata/enron-people"
	"github.com/grokify/commgraph/entity"
	"github.com/grokify/goauth/scim"
)

// LoadEnronPeople loads all known Enron employees into the resolver.
// This pre-populates the resolver with canonical identities and their aliases,
// enabling proper merging when the same person uses multiple email addresses.
// It loads both the curated SCIM data and the custodians JSON data.
func (r *SCIMResolver) LoadEnronPeople() int {
	count := 0

	// Load curated SCIM data (has detailed aliases for key people)
	peopleSet := enronpeople.NewPeopleSet()
	count += r.LoadSCIMUserSet(peopleSet)

	// Load custodians from JSON (148 employees with auto-generated aliases)
	custodiansSet, err := enronpeople.NewCustodiansUserSet()
	if err == nil {
		count += r.LoadSCIMUserSet(custodiansSet)
	}

	return count
}

// LoadSCIMUserSet loads a SCIM UserSet into the resolver.
func (r *SCIMResolver) LoadSCIMUserSet(userSet scim.UserSet) int {
	count := 0
	for _, user := range userSet.Users {
		actor := SCIMUserToActor(user)
		r.LoadActor(actor)
		count++
	}
	return count
}

// SCIMUserToActor converts a SCIM user to an Actor.
func SCIMUserToActor(user scim.User) *entity.Actor {
	// Extract emails
	var emails []string
	var primaryEmail string
	for _, email := range user.Emails {
		emailLower := strings.ToLower(strings.TrimSpace(email.Value))
		if emailLower != "" {
			emails = append(emails, emailLower)
			if email.Primary {
				primaryEmail = emailLower
			}
		}
	}
	if primaryEmail == "" && len(emails) > 0 {
		primaryEmail = emails[0]
	}

	// Generate ID from username (matches maildir folder name)
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
	displayName := user.DisplayNameAny()

	return &entity.Actor{
		ID:           entity.ActorID(id),
		DisplayName:  displayName,
		Emails:       emails,
		PrimaryEmail: primaryEmail,
		ExternalID:   user.ExternalID,
		Internal:     true, // Enron employees are internal
		Department:   department,
		Title:        user.Title,
		Timezone:     user.Timezone,
		Metadata: map[string]string{
			"username":  user.UserName,
			"nickname":  user.NickName,
			"profile":   user.ProfileURL,
			"user_type": user.UserType,
			"locale":    user.Locale,
			"given":     user.Name.GivenName,
			"family":    user.Name.FamilyName,
			"middle":    user.Name.MiddleName,
			"suffix":    user.Name.HonorificSuffix,
		},
	}
}
